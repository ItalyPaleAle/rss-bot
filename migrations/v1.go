package migrations

import (
	"github.com/ItalyPaleAle/rss-bot/db"
)

func V1() error {
	DB := db.GetDB()
	sqlStmt := `
CREATE TABLE IF NOT EXISTS feeds (
	feed_id integer primary key autoincrement,
	feed_url text not null,
	feed_last_modified timestamp not null,
	feed_etag text not null,
	feed_last_post_title text not null,
	feed_last_post_link text not null,
	feed_last_post_date timestamp not null
);
CREATE UNIQUE INDEX IF NOT EXISTS feeds_feed_url ON feeds (feed_url);
CREATE TABLE IF NOT EXISTS subscriptions (
	subscription_id integer primary key autoincrement,
	feed_id integer not null,
	chat_id integer not null
);
CREATE INDEX IF NOT EXISTS subscriptions_chat_id ON subscriptions (chat_id);
CREATE TABLE IF NOT EXISTS migrations (
	version integer
);
`

	_, err := DB.Exec(sqlStmt)
	if err != nil {
		return err
	}
	return nil
}
