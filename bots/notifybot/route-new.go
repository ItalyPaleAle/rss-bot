package main

import (
	"crypto/sha256"
	"time"

	"github.com/jmoiron/sqlx"
	nanoid "github.com/matoous/go-nanoid/v2"

	"github.com/ItalyPaleAle/rss-bot/bots/notifybot/db"
	pb "github.com/ItalyPaleAle/rss-bot/model"
)

// Maximum number of webhooks per each chat
const MaxWebhooksPerChat = 10

// Route for the "new webhook" command
func (nb *NotifyBot) routeNew(m *pb.InMessage) {
	DB := db.GetDB()

	// Count how many webhooks this chat is subscribed to, and limit that to MaxWebhooksPerChat
	count := struct {
		Count int64 `db:"count"`
	}{}
	err := sqlx.Get(DB, &count, "SELECT COUNT(webhook_id) AS count FROM webhooks WHERE chat_id = ?", m.ChatId)
	if err != nil {
		nb.log.Printf("Error counting webhooks for chat %d: %s\n", m.ChatId, err.Error())
		return
	}
	if count.Count >= MaxWebhooksPerChat {
		_, err = nb.client.RespondToCommand(m, "Sorry, this chat has already reached the maximum number of webhooks and I can't add another one")
		if err != nil {
			nb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
			return
		}
		return
	}

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
	_, err = DB.Exec("INSERT INTO webhooks (webhook_id, webhook_key, webhook_created, chat_id) VALUES (?, ?, ?, ?)", webhookId, webhookKeyHash[:], time.Now().Unix(), m.ChatId)
	if err != nil {
		nb.log.Printf("Error storing new webhook for chat %d: %s\n", m.ChatId, err.Error())
		return
	}

	// Respond with the key
	// TODO: Write full URL
	_, err = nb.client.RespondToCommand(m, &pb.OutTextMessage{
		Text: `Here's the webhook I've created for you:
ID: <code>` + webhookId + `</code>
URL: <code>https://localhost:8080/webhook/` + webhookId + `</code>
Access token: <code>` + webhookKey + `</code>`,
		ParseMode: pb.ParseMode_HTML,
	})
	if err != nil {
		nb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		return
	}
}
