package main

import (
	"fmt"

	pb "github.com/ItalyPaleAle/rss-bot/model"
)

// Route for the "list feed(s)" command
func (fb *FeedBot) routeList(m *pb.InMessage) {
	// Get the list of subscriptions
	feeds, err := fb.feeds.ListSubscriptions(m.ChatId)
	if err != nil {
		_, err := fb.client.RespondToCommand(m, "An internal error occurred")
		if err != nil {
			// Log errors only
			fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		}
		return
	}

	// Build the response
	if len(feeds) == 0 {
		_, err := fb.client.RespondToCommand(m, "I can't find any feed this chat is subscribed to")
		if err != nil {
			// Log errors only
			fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		}
		return
	}

	out := "Here's the list of feeds this chat is subscribed to:\n"
	for i, f := range feeds {
		out += fmt.Sprintf("%d: <code>%s</code>\n", (i + 1), f.Url)
	}
	_, err = fb.client.RespondToCommand(m, &pb.OutMessage{
		Content: &pb.OutMessage_Text{
			Text: &pb.OutTextMessage{
				Text:      out,
				ParseMode: pb.ParseMode_HTML,
			},
		},
		Options: &pb.OutMessageOptions{
			DisableWebPagePreview: true,
		},
	})
	if err != nil {
		// Log errors only
		fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
	}
}
