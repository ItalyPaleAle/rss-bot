package migrations

import (
	"database/sql"
	"fmt"

	"github.com/ItalyPaleAle/rss-bot/db"
)

func V3() error {
	DB := db.GetDB()

	// Get the version
	res := &struct {
		Version int
	}{}
	err := DB.Get(res, "SELECT * FROM migrations WHERE ROWID = 0")
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	version := res.Version

	// Update to version 3 if needed
	if version < 3 {
		fmt.Println("Migrating database to version 3")
		sqlStmt := `
ALTER TABLE feeds ADD COLUMN feed_last_post_photo text not null default "";
UPDATE migrations SET version = 3 WHERE ROWID = 0;
`

		_, err := DB.Exec(sqlStmt)
		if err != nil {
			return err
		}
	}
	return nil
}
