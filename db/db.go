package db

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	db *sqlx.DB
)

func GetDB() *sqlx.DB {
	return db
}

func ConnectDB() *sqlx.DB {
	var err error
	log.Debug("Connecting to the database")
	db, err = sqlx.Open("sqlite3", "file:"+viper.GetString("db_path")+"?cache=shared")

	// Check error for database connection
	if err != nil {
		panic(err)
	}

	return db
}
