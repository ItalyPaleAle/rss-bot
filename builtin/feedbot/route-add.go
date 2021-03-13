package feedbot

import (
	"regexp"

	"github.com/ItalyPaleAle/rss-bot/feeds"
	pb "github.com/ItalyPaleAle/rss-bot/proto"
)

var routeAddMatch = regexp.MustCompile("(?i)^add feed (.*)")

// Handles /add commands
func (b *FeedBot) routeAdd(m *pb.InMessage) {
	// Get the URL
	match := routeAddMatch.FindStringSubmatch(m.Text)
	if len(match) < 2 {
		b.manager.RespondToCommand(m, "I didn't understand this \"add feed\" message - is the URL missing?")
		return
	}
	url := match[1]

	// Send a message that we're working on it
	sent, err := b.manager.RespondToCommand(m, "Working on it…")
	if err != nil {
		b.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		return
	}

	// Add the subscription
	post, err := b.feeds.AddSubscription(url, m.ChatId)
	if err != nil {
		if err == feeds.ErrAlreadySubscribed {
			err := b.manager.EditTextMessage(&pb.EditTextMessage{
				Message: sent,
				Text: &pb.OutTextMessage{
					Text: "You've already subscribed this chat to the feed",
				},
			})
			if err != nil {
				// Log errors only
				b.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
			}
		} else {
			err := b.manager.EditTextMessage(&pb.EditTextMessage{
				Message: sent,
				Text: &pb.OutTextMessage{
					Text: "An internal error occurred",
				},
			})
			if err != nil {
				// Log errors only
				b.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
			}
		}
		return
	}

	err = b.manager.EditTextMessage(&pb.EditTextMessage{
		Message: sent,
		Text: &pb.OutTextMessage{
			Text: "I've added the feed to this channel. Here is the last post they published:",
		},
	})
	if err != nil {
		b.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		return
	}
	b.sendFeedUpdate(&feeds.UpdateMessage{
		Post:   *post,
		ChatId: m.ChatId,
	})
}
