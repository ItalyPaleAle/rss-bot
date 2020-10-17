package main

import (
	"github.com/0x111/telegram-rss-bot/bot"
	"github.com/0x111/telegram-rss-bot/conf"
	"github.com/0x111/telegram-rss-bot/db"
	"github.com/0x111/telegram-rss-bot/migrations"
)

func main() {
	// Load config
	conf.LoadConfig()

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
