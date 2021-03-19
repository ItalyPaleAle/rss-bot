package migrations

import (
	"database/sql"
	"fmt"

	"github.com/ItalyPaleAle/rss-bot/db"
)

var version = 0

// Migrate runs the migration suite
func Migrate() {
	// A rather makeshift solution, but it works for our simple scenario
	DB := db.GetDB()

	// Check if the "migrations" table exists
	resTable := &struct {
		Name string
	}{}
	err := DB.Get(resTable, "SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'migrations'")
	if err == sql.ErrNoRows {
		// Migrations table does not exist
		_, err := DB.Exec("CREATE TABLE migrations (version integer);")
		if err != nil {
			panic(fmt.Sprintln("Error creating the migrations table", err))
		}
	} else if err != nil {
		panic(fmt.Sprintln("Error getting the list of tables", err))
	}

	// Get the version of the schema
	resVersion := &struct {
		Version int
	}{}
	err = DB.Get(resVersion, "SELECT * FROM migrations WHERE ROWID = 0")
	if err == sql.ErrNoRows {
		version = 0
	} else if err != nil {
		panic(fmt.Sprintln("Error getting the schema version", err))
	} else {
		version = resVersion.Version
	}

	// Perform all migrations
	if version < 1 {
		fmt.Println("Migrating database to version 1")
		if err := V1(); err != nil {
			panic(fmt.Sprintln("Error migrating the database to V1", err))
		}
	}
	if version < 2 {
		fmt.Println("Migrating database to version 2")
		if err := V2(); err != nil {
			panic(fmt.Sprintln("Error migrating the database to V2", err))
		}
	}
	if version < 3 {
		fmt.Println("Migrating database to version 3")
		if err := V3(); err != nil {
			panic(fmt.Sprintln("Error migrating the database to V3", err))
		}
	}
	if version < 4 {
		fmt.Println("Migrating database to version 4")
		if err := V4(); err != nil {
			panic(fmt.Sprintln("Error migrating the database to V4", err))
		}
	}
}
