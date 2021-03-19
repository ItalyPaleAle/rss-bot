package db

// Model for the webhooks table
type Webhook struct {
	ID     string `db:"webhook_id"`
	Key    []byte `db:"webhook_key"`
	ChatID int64  `db:"chat_id"`
}
