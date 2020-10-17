package bot

import (
	tb "gopkg.in/tucnak/telebot.v2"
)

// Handles /help commands
func (b *RSSBot) handleHelp(m *tb.Message) {
	// Send the help message
	b.bot.Send(m.Sender, `
Avaliable commands:
/add <URL> - Subscribe to a new feed for this channel
/list - List all subscribed feeds for this channel
/delete <ID> - Remove a feed subscription
`)
}
