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
	// Defaults
	viper.SetDefault("TelegramAuthToken", "")
	viper.SetDefault("TelegramAPIDebug", false)
	viper.SetDefault("DBPath", "./bot.db")
	viper.SetDefault("FeedUpdateInterval", 600)
	viper.SetDefault("AllowedUsers", nil)

	// Env
	viper.SetEnvPrefix("BOT")
	viper.AutomaticEnv()

	// Config file
	viper.SetConfigName("bot-config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.rss-bot")
	viper.AddConfigPath("/etc/rss-bot")

	// Read the config
	err := viper.ReadInConfig()
	if err != nil {
		// Ignore errors if the config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(fmt.Sprintf("Fatal error config file: %s\n", err))
		}
	}
}
