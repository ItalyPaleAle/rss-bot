package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	db *sql.DB
)

func GetDB() *sql.DB {
	return db
}

func ConnectDB() *sql.DB {
	var err error
	log.Debug("Connecting to the database")
	db, err = sql.Open("sqlite3", "file:"+viper.GetString("db_path")+"?cache=shared")

	// Check error for database connection
	if err != nil {
		panic(err)
	}

	return db
}
