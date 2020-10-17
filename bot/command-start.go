package bot

import (
	tb "gopkg.in/tucnak/telebot.v2"
)

// Handles /start commands
func (b *RSSBot) handleStart(m *tb.Message) {
	// Send the welcome message
	b.bot.Send(m.Sender, "Welcome to the RSS bot!")

	// Send the help message too
	b.handleHelp(m)
}
