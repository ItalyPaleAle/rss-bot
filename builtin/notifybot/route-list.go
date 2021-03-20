package notifybot

import (
	"fmt"

	"github.com/ItalyPaleAle/rss-bot/db"
	pb "github.com/ItalyPaleAle/rss-bot/proto"
)

// Route for the "list webhook(s)" command
func (nb *NotifyBot) routeList(m *pb.InMessage) {
	// Get the webhooks from the database
	DB := db.GetDB()
	list := []db.Webhook{}
	err := DB.Select(&list, "SELECT * FROM webhooks WHERE chat_id = ?", m.ChatId)
	if err != nil {
		nb.log.Printf("Error retrieving webhooks for chat %d: %s\n", m.ChatId, err.Error())
		return
	}

	// Build the response
	if len(list) == 0 {
		_, err := nb.manager.RespondToCommand(m, "I can't find any webhook for this chat")
		if err != nil {
			// Log errors only
			nb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		}
		return
	}

	out := "Here's the list of webhooks for this chat:\n"
	for i, v := range list {
		out += fmt.Sprintf("%d: <code>%s</code>\n", (i + 1), v.ID)
	}
	_, err = nb.manager.RespondToCommand(m, &pb.OutMessage{
		Content: &pb.OutMessage_Text{
			Text: &pb.OutTextMessage{
				Text:      out,
				ParseMode: pb.ParseMode_HTML,
			},
		},
		DisableWebPagePreview: true,
	})
	if err != nil {
		// Log errors only
		nb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
	}
}
