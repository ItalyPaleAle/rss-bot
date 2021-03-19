package migrations

import (
	"github.com/ItalyPaleAle/rss-bot/db"
)

func V3() error {
	DB := db.GetDB()
	_, err := DB.Exec(`
ALTER TABLE feeds ADD COLUMN feed_last_post_photo text not null default "";

UPDATE migrations SET version = 3 WHERE ROWID = 0;
`)
	if err != nil {
		return err
	}
	version = 3
	return nil
}
