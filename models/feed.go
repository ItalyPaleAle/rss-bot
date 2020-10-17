package models

import "time"

// Model for the feeds table
type Feed struct {
	ID            int64     `db:"feed_id"`
	Url           string    `db:"feed_url"`
	LastPostTitle string    `db:"feed_last_post_title"`
	LastPostLink  string    `db:"feed_last_post_link"`
	LastPostDate  time.Time `db:"feed_last_post_date"`
}
