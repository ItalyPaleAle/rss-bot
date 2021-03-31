package main

import (
	"regexp"
	"strconv"

	pb "github.com/ItalyPaleAle/rss-bot/model"
)

var routeRemoveMatch = regexp.MustCompile("(?i)^(remove|delete) feed (.*)")

// Route for the "remove feed" command
func (fb *FeedBot) routeRemove(m *pb.InMessage) {
	// Get the arg
	match := routeRemoveMatch.FindStringSubmatch(m.Text)
	if len(match) < 3 {
		_, err := fb.client.RespondToCommand(m, "I didn't understand this \"remove feed\" message - is the feed ID or URL missing?")
		if err != nil {
			// Log errors only
			fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		}
		return
	}

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

	// Check if we have an ID or a feed URL
	feedIdx, err := strconv.ParseInt(match[2], 10, 64)
	if err != nil || feedIdx <= 0 || feedIdx > int64(len(feeds)) {
		// We might have a feed URL
		feedIdx = -1
		for i, v := range feeds {
			if v.Url == match[2] {
				feedIdx = int64(i)
				break
			}
		}

		if feedIdx == -1 {
			_, err := fb.client.RespondToCommand(m, "I cannot find the feed you're trying to delete!")
			if err != nil {
				// Log errors only
				fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
			}
			return
		}
	} else {
		// Users type indexes starting from 1, so decrease this
		feedIdx--
	}

	// Delete the subscription
	err = fb.feeds.DeleteSubscription(feeds[feedIdx].ID, m.ChatId)
	if err != nil {
		_, err := fb.client.RespondToCommand(m, "An internal error occurred")
		if err != nil {
			// Log errors only
			fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		}
		return
	}

	// All good
	_, err = fb.client.RespondToCommand(m, &pb.OutMessage{
		Content: &pb.OutMessage_Text{
			Text: &pb.OutTextMessage{
				Text: "Ok, I've removed the subscription to " + feeds[feedIdx].Url,
			},
		},
		Options: &pb.OutMessageOptions{
			DisableWebPagePreview: true,
		},
	})
	if err != nil {
		// Log errors only
		fb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		return
	}
}
