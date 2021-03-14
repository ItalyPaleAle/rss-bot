package feedbot

import (
	"regexp"

	"github.com/ItalyPaleAle/rss-bot/feeds"
	pb "github.com/ItalyPaleAle/rss-bot/proto"
)

var routeAddMatch = regexp.MustCompile("(?i)^add feed (.*)")

// Route for the "add feed" command
func (fb *FeedBot) routeAdd(m *pb.InMessage) {
	// Get the URL
	match := routeAddMatch.FindStringSubmatch(m.Text)
	if len(match) < 2 {
		fb.manager.RespondToCommand(m, "I didn't understand this \"add feed\" message - is the URL missing?")
		return
	}
	url := match[1]

	// Send a message that we're working on it
	sent, err := fb.manager.RespondToCommand(m, "Working on it…")
	if err != nil {
		fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		return
	}

	// Add the subscription
	post, err := fb.feeds.AddSubscription(url, m.ChatId)
	if err != nil {
		if err == feeds.ErrAlreadySubscribed {
			err := fb.manager.EditTextMessage(&pb.EditTextMessage{
				Message: sent,
				Text: &pb.OutTextMessage{
					Text: "You've already subscribed this chat to the feed",
				},
			})
			if err != nil {
				// Log errors only
				fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
			}
		} else {
			// Log errors and then send a message
			fb.log.Printf("Error while adding feed to chat %d: %s\n", m.ChatId, err.Error())

			err := fb.manager.EditTextMessage(&pb.EditTextMessage{
				Message: sent,
				Text: &pb.OutTextMessage{
					Text: "An internal error occurred",
				},
			})
			if err != nil {
				// Log errors only
				fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
			}
		}
		return
	}

	err = fb.manager.EditTextMessage(&pb.EditTextMessage{
		Message: sent,
		Text: &pb.OutTextMessage{
			Text: "I've added the feed to this channel. Here is the last post they published:",
		},
	})
	if err != nil {
		fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		return
	}
	fb.sendFeedUpdate(&feeds.UpdateMessage{
		Post:   *post,
		ChatId: m.ChatId,
	})
}
