package migrations

import (
	"database/sql"
	"fmt"

	"github.com/ItalyPaleAle/rss-bot/db"
)

func V2() error {
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

	// Update to version 2 if needed
	if version < 2 {
		fmt.Println("Migrating database to version 2")
		sqlStmt := `
ALTER TABLE feeds ADD COLUMN feed_title text not null default "";
INSERT OR REPLACE INTO migrations (ROWID, version) VALUES (0, 2);
`

		_, err := DB.Exec(sqlStmt)
		if err != nil {
			return err
		}
	}
	return nil
}
