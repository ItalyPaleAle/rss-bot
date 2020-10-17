package db

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

var (
	db *sqlx.DB
)

func GetDB() *sqlx.DB {
	return db
}

func ConnectDB() *sqlx.DB {
	// Init the singleton
	var err error
	db, err = sqlx.Open("sqlite3", "file:"+viper.GetString("db_path")+"?cache=shared")
	if err != nil {
		panic(err)
	}

	return db
}
