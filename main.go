package main

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/ItalyPaleAle/rss-bot/bot"
	"github.com/ItalyPaleAle/rss-bot/db"
	"github.com/ItalyPaleAle/rss-bot/migrations"
)

func main() {
	// Load config
	loadConfig()

	// Connect to DB and migrate to the latest version
	dbc := db.ConnectDB()
	defer dbc.Close()
	migrations.Migrate()

	// Create the bot
	b := &bot.RSSBot{}
	err := b.Init()
	if err != nil {
		panic(err)
	}

	// Start the bot - this is a blocking call
	err = b.Start()
	if err != nil {
		panic(err)
	}
}

func loadConfig() {
	viper.SetConfigName("bot-config")
	viper.AddConfigPath("$HOME/.telegram-rss-bot")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./.telegram-rss-bot")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Sprintf("Fatal error config file: %s\n", err))
	}
}
