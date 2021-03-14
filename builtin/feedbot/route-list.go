package feedbot

import (
	"fmt"

	pb "github.com/ItalyPaleAle/rss-bot/proto"
)

// Route for the "list feed(s)" command
func (fb *FeedBot) routeList(m *pb.InMessage) {
	// Get the list of subscriptions
	feeds, err := fb.feeds.ListSubscriptions(m.ChatId)
	if err != nil {
		_, err := fb.manager.RespondToCommand(m, "An internal error occurred")
		if err != nil {
			// Log errors only
			fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		}
		return
	}

	// Build the response
	if len(feeds) == 0 {
		_, err := fb.manager.RespondToCommand(m, "This chat is not subscribed to any feed")
		if err != nil {
			// Log errors only
			fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		}
		return
	}

	out := "Here's the list of feeds this chat is subscribed to:\n"
	for i, f := range feeds {
		out += fmt.Sprintf("%d: %s\n", (i + 1), f.Url)
	}
	_, err = fb.manager.RespondToCommand(m, &pb.OutMessage{
		Content: &pb.OutMessage_Text{
			Text: &pb.OutTextMessage{
				Text: out,
			},
		},
		DisableWebPagePreview: true,
	})
	if err != nil {
		// Log errors only
		fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
	}
}
