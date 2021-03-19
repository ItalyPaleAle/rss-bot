package migrations

import (
	"github.com/ItalyPaleAle/rss-bot/db"
)

func V1() error {
	DB := db.GetDB()
	_, err := DB.Exec(`
CREATE TABLE feeds (
	feed_id integer primary key autoincrement,
	feed_url text not null,
	feed_last_modified timestamp not null,
	feed_etag text not null,
	feed_last_post_title text not null,
	feed_last_post_link text not null,
	feed_last_post_date timestamp not null
);

CREATE UNIQUE INDEX feeds_feed_url ON feeds (feed_url);

CREATE TABLE subscriptions (
	subscription_id integer primary key autoincrement,
	feed_id integer not null,
	chat_id integer not null
);

CREATE INDEX subscriptions_chat_id ON subscriptions (chat_id);

INSERT INTO migrations (ROWID, version) VALUES (0, 1);
`)
	if err != nil {
		return err
	}
	version = 1
	return nil
}
