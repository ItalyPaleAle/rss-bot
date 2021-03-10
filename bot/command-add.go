package bot

import (
	"fmt"

	"github.com/ItalyPaleAle/rss-bot/feeds"
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

	// Send a message that we're working on it
	wm, _ := b.respondToCommand(m, "Working on it…")

	// Add the subscription
	post, err := b.feeds.AddSubscription(url, m.Chat.ID)
	if err != nil {
		if err == feeds.ErrAlreadySubscribed {
			b.bot.Edit(wm, "This chat is already subscribed to the feed")
		} else {
			b.bot.Edit(wm, "An internal error occurred")
		}
		return
	}

	b.bot.Edit(wm, fmt.Sprintf("The feed with URL %s was successfully added to this channel. Here is the last post published:", url), &tb.SendOptions{
		DisableWebPagePreview: true,
	})
	b.bot.Send(m.Sender, b.formatUpdateMessage(&feeds.UpdateMessage{
		Post: *post,
	}), &tb.SendOptions{
		ParseMode:             tb.ModeHTML,
		DisableWebPagePreview: true,
	})
}
