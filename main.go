package main

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/ItalyPaleAle/rss-bot/bot"
	"github.com/ItalyPaleAle/rss-bot/builtin/feedbot"
	"github.com/ItalyPaleAle/rss-bot/db"
	"github.com/ItalyPaleAle/rss-bot/migrations"
)

func main() {
	// Ensure that the app is running on a system with 64-bit integers
	if int64(int(1<<60)) != int64(1<<60) {
		panic("This app should only be executed on a 64-bit system")
	}

	// Load config
	loadConfig()

	// Connect to DB and migrate to the latest version
	dbc := db.ConnectDB()
	defer dbc.Close()
	migrations.Migrate()

	// Create the bot
	b := &bot.BotManager{}
	err := b.Init()
	if err != nil {
		panic(err)
	}

	// Add built-in features
	{
		// FeedBot: RSS and Atom feeds
		feature := &feedbot.FeedBot{}
		err := feature.Init(b)
		if err != nil {
			panic(err)
		}
		err = feature.Start()
		if err != nil {
			panic(err)
		}
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
	viper.SetDefault("AllowedUsers", nil)
	viper.SetDefault("DBPath", "./bot.db")
	viper.SetDefault("FeedUpdateInterval", 600)

	// Env
	viper.SetEnvPrefix("BOT")
	viper.AutomaticEnv()

	// Config file
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.bot")
	viper.AddConfigPath("/etc/bot")

	// Read the config
	err := viper.ReadInConfig()
	if err != nil {
		// Ignore errors if the config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(fmt.Sprintf("Fatal error config file: %s\n", err))
		}
	}
}
