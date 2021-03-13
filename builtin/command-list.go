package bot

import (
	"fmt"

	tb "gopkg.in/tucnak/telebot.v2"
)

// Handles /list commands
func (b *FeedBot) handleList(m *tb.Message) {
	// Get the list of subscriptions
	feeds, err := b.feeds.ListSubscriptions(m.Chat.ID)
	if err != nil {
		b.respondToCommand(m, "An internal error occurred")
		return
	}

	// Build the response
	if len(feeds) == 0 {
		b.respondToCommand(m, "This chat is not subscribed to any feed")
		return
	}

	out := "Here's the list of feeds this chat is subscribed to:\n"
	for i, f := range feeds {
		out += fmt.Sprintf("%d: %s\n", (i + 1), f.Url)
	}
	b.respondToCommand(m, out)
}
