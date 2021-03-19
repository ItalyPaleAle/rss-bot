package migrations

import (
	"github.com/ItalyPaleAle/rss-bot/db"
)

func V4() error {
	DB := db.GetDB()
	_, err := DB.Exec(`
CREATE TABLE webhooks (
	webhook_id text not null,
	webhook_key blob not null,
	chat_id integer not null
);

CREATE UNIQUE INDEX webhooks_webhook_id ON webhooks (webhook_id);

UPDATE migrations SET version = 4 WHERE ROWID = 0;
`)
	if err != nil {
		return err
	}
	version = 4
	return nil
}
