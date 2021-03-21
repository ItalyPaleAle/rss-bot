package db

// Model for the webhooks table
type Webhook struct {
	ID      string `db:"webhook_id"`
	Key     []byte `db:"webhook_key"`
	Created int64  `db:"webhook_created"`
	ChatID  int64  `db:"chat_id"`
}
