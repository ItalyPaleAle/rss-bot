package db

import (
	"fmt"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"

	"github.com/ItalyPaleAle/rss-bot/utils"
)

var (
	connection *sqlx.DB
)

func GetDB() *sqlx.DB {
	return connection
}

func ConnectDB() *sqlx.DB {
	// Check if the path is set
	dbPath := viper.GetString("DBPath")
	if dbPath == "" {
		panic("Database path is empty")
	}

	// Ensure the folder exists
	dbPath, err := filepath.Abs(dbPath)
	if err != nil {
		panic(fmt.Sprintln("Invalid database path", err))
	}
	dbDir := filepath.Dir(dbPath)
	utils.EnsureFolder(dbDir)
	if err != nil {
		panic(fmt.Sprintln("Could not create the folder for the database", err))
	}

	// Init the singleton
	connection, err = sqlx.Open("sqlite3", "file:"+dbPath+"?cache=shared")
	if err != nil {
		panic(err)
	}

	return connection
}
