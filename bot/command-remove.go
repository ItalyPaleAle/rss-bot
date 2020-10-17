package bot

import (
	"fmt"
	"strconv"

	tb "gopkg.in/tucnak/telebot.v2"
)

// Handles /remove commands
func (b *RSSBot) handleRemove(m *tb.Message) {
	// Get args
	args := GetArgs(m.Payload)
	if len(args) != 1 {
		b.respondToCommand(m, "Invalid arguments: need \"/remove <id>\"")
		return
	}
	id, err := strconv.Atoi(args[0])
	if err != nil || id < 1 {
		b.respondToCommand(m, "Invalid arguments: need \"/remove <id>\"")
		return
	}

	// Get the list of subscriptions
	feeds, err := b.feeds.ListSubscriptions(m.Chat.ID)
	if err != nil {
		b.respondToCommand(m, "An internal error occurred")
		return
	}

	// Check if the feed exists
	if id > len(feeds) {
		b.respondToCommand(m, "Subscription not found")
		return
	}

	// Ask for confirmation
	r := &tb.ReplyMarkup{
		// Reply keyboard
		InlineKeyboard: [][]tb.InlineButton{
			{tb.InlineButton{Text: "Confirm", Data: fmt.Sprintf("confirm-remove/%d", feeds[id-1].ID)}},
			{tb.InlineButton{Text: "Cancel", Unique: "cancel"}},
		},
		// Hide the keyboard after using it once
		OneTimeKeyboard: true,
		// In a group, allow responses only from the user who submitted the request
		Selective: true,
	}
	opts := &tb.SendOptions{
		ReplyMarkup: r,
	}
	b.respondToCommand(m, fmt.Sprintf("Are you sure you want to remove the feed %s?", feeds[id-1].Url), opts)
}

// Handles the callbacks with "confirm-remove" action
func (b *RSSBot) callbackConfirmRemove(cb *tb.Callback, userData string) {
	// Get the feed ID to delete
	feedId, err := strconv.Atoi(userData)
	if err != nil || feedId < 1 {
		b.log.Println("Invalid feedId in confirm-remove callback", err)
		b.bot.Send(cb.Message.Chat, "An internal error occurred")
		return
	}

	// Delete the subscription
	err = b.feeds.DeleteSubscription(int64(feedId), cb.Message.Chat.ID)
	if err != nil {
		// Error is already logged
		b.bot.Send(cb.Message.Chat, "An internal error occurred")
		return
	}

	// Update the message
	_, err = b.bot.Edit(cb.Message, "Done, I've removed the subscription")
	if err != nil {
		b.log.Println("Error while editing message:", err)
		return
	}
}
