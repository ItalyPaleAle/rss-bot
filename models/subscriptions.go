package models

// Model for the subscriptions table
type Subscription struct {
	ID     int64 `db:"subscription_id"`
	FeedID int64 `db:"feed_id"`
	ChatID int64 `db:"chat_id"`
}
