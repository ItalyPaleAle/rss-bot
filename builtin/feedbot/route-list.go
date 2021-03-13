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
		fb.manager.RespondToCommand(m, "An internal error occurred")
		return
	}

	// Build the response
	if len(feeds) == 0 {
		fb.manager.RespondToCommand(m, "This chat is not subscribed to any feed")
		return
	}

	out := "Here's the list of feeds this chat is subscribed to:\n"
	for i, f := range feeds {
		out += fmt.Sprintf("%d: %s\n", (i + 1), f.Url)
	}
	fb.manager.RespondToCommand(m, out)
}
