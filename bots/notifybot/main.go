package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"

	"github.com/ItalyPaleAle/rss-bot/bots/notifybot/db"
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
	db.Migrate()

	// Create the NotifyBot object
	bot := &NotifyBot{}
	err := bot.Init()
	if err != nil {
		panic(err)
	}
	err = bot.Start()
	if err != nil {
		panic(err)
	}

	// Handle graceful shutdown on SIGINT, SIGTERM and SIGQUIT
	stopSigCh := make(chan os.Signal, 1)
	signal.Notify(stopSigCh,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	// Wait for the shutdown signal then stop the bot and the server
	<-stopSigCh
	err = bot.Stop()
	if err != nil {
		panic(err)
	}
}

func loadConfig() {
	// Defaults
	viper.SetDefault("DBPath", "./bot.db")

	// Env
	viper.SetEnvPrefix("NOTIFYBOT")
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
