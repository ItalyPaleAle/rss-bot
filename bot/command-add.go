package bot

import (
	"fmt"

	"github.com/0x111/telegram-rss-bot/feeds"
	tb "gopkg.in/tucnak/telebot.v2"
)

// Handles /add commands
func (b *RSSBot) handleAdd(m *tb.Message) {
	// Get args
	args := GetArgs(m.Payload)
	if len(args) != 1 {
		b.respondToCommand(m, "Invalid arguments: need \"/add <url>\"")
		return
	}
	url := args[0]
	if url == "" {
		b.respondToCommand(m, "Invalid arguments: need \"/add <url>\"")
		return
	}

	// Add the subscription
	post, err := b.feeds.AddSubscription(url, m.Chat.ID)
	if err != nil {
		if err == feeds.ErrAlreadySubscribed {
			b.respondToCommand(m, "This chat is already subscribed to the feed")
		} else {
			b.respondToCommand(m, "An internal error occurred")
		}
		return
	}

	b.respondToCommand(m, fmt.Sprintf("The feed with URL %s wa successfully added to this channel. Here is the last post published:", url))
	b.respondToCommand(m, b.formatUpateMessage(&feeds.UpdateMessage{
		Post: *post,
	}))
}
