package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"

	"github.com/ItalyPaleAle/rss-bot/core/bot"
	"github.com/ItalyPaleAle/rss-bot/core/server"
)

func main() {
	// Ensure that the app is running on a system with 64-bit integers
	if int64(int(1<<60)) != int64(1<<60) {
		panic("This app should only be executed on a 64-bit system")
	}

	// Load config
	loadConfig()

	// Execution context
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// Create the bot
	b := &bot.BotManager{
		Ctx: ctx,
	}
	err := b.Init()
	if err != nil {
		panic(err)
	}

	// Start the bot in a background goroutine
	go b.Start()

	// Start the gRPC server in a background goroutine
	srv := &server.RPCServer{
		Ctx: ctx,
	}
	srv.Init(b)
	go srv.Start()

	// Handle graceful shutdown on SIGINT, SIGTERM and SIGQUIT
	stopSigCh := make(chan os.Signal, 1)
	signal.Notify(stopSigCh,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	// Wait for the shutdown signal then stop the bot and the server
	<-stopSigCh
	b.Stop()
	srv.Stop()
}

func loadConfig() {
	// Defaults
	viper.SetDefault("TelegramAuthToken", "")
	viper.SetDefault("TelegramAPIDebug", false)
	viper.SetDefault("AllowedUsers", nil)

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
