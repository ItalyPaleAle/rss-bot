package notifybot

import (
	"crypto/sha256"

	"github.com/ItalyPaleAle/rss-bot/db"
	pb "github.com/ItalyPaleAle/rss-bot/proto"

	nanoid "github.com/matoous/go-nanoid/v2"
)

// Route for the "new webhook" command
func (nb *NotifyBot) routeNew(m *pb.InMessage) {
	// Create a new webhook ID and secret
	webhookId, err := nanoid.New(21)
	if err != nil {
		nb.log.Printf("Error generating the webhook ID for chat %d: %s\n", m.ChatId, err.Error())
		return
	}
	webhookKey, err := nanoid.New(21)
	if err != nil {
		nb.log.Printf("Error generating the webhook secret for chat %d: %s\n", m.ChatId, err.Error())
		return
	}
	webhookKey = "SK_" + webhookKey

	// Calculate the hash of the secret
	// This uses only 1 round of SHA-256 which is not ideal, but it should be fine in this case as the nanoid has 126 bits of entropy already
	webhookKeyHash := sha256.Sum256([]byte(webhookKey))

	// Insert in the database
	DB := db.GetDB()
	_, err = DB.Exec("INSERT INTO webhooks (webhook_id, webhook_key, chat_id) VALUES (?, ?, ?)", webhookId, webhookKeyHash[:], m.ChatId)
	if err != nil {
		nb.log.Printf("Error storing new webhook for chat %d: %s\n", m.ChatId, err.Error())
		return
	}

	// Respond with the key
	// TODO: Write full URL
	_, err = nb.manager.RespondToCommand(m, &pb.OutTextMessage{
		Text:      "Here's the webhook I've created for you:\nID: `" + webhookId + "`\nURL: `https://localhost:8080/webhook/" + webhookId + "`\nAccess token: `" + webhookKey + "`",
		ParseMode: pb.ParseMode_MARKDOWN_V2,
	},
	)
	if err != nil {
		nb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		return
	}
}
