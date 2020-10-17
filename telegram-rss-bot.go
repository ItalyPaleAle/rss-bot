package main

import (
	"github.com/0x111/telegram-rss-bot/chans"
	"github.com/0x111/telegram-rss-bot/commands"
	"github.com/0x111/telegram-rss-bot/conf"
	"github.com/0x111/telegram-rss-bot/db"
	"github.com/0x111/telegram-rss-bot/migrations"

	tgbotapi "github.com/dilfish/telegram-bot-api-up"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	// Load config
	conf.LoadConfig()

	// Connect to DB
	dbc := db.ConnectDB()
	defer dbc.Close()

	var err error

	// Init the bot API
	Bot, err := tgbotapi.NewBotAPI(viper.GetString("telegram_auth_key"))
	if err != nil {
		log.Panic(err)
	}

	Bot.Debug = viper.GetBool("telegram_api_debug")

	log.Debug("Authorized on account ", Bot.Self.UserName)

	// create basic database structure
	migrations.Migrate()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// read rss data from channels
	go func() {
		chans.FeedUpdates()
	}()

	// post rss data to channels
	go func() {
		chans.FeedPosts(Bot)
	}()

	// Check if we're allowing certain users to use the bot only
	var allowedUsers map[int]bool
	if uids := viper.GetIntSlice("allowed_users"); len(uids) > 0 {
		// Create a map so lookups are faster
		allowedUsers = make(map[int]bool, len(uids))
		for i := 0; i < len(uids); i++ {
			allowedUsers[uids[i]] = true
		}
	}

	// read feed updates from the Telegram API
	updates, err := Bot.GetUpdatesChan(u)
	for update := range updates {
		// if the message is empty, we do not need to handle anything
		if update.Message == nil {
			continue
		}

		// Check if we're restricting to some users only
		if allowedUsers != nil &&
			(update.Message.From == nil || update.Message.From.ID == 0 || !allowedUsers[update.Message.From.ID]) {
			continue
		}

		// allow only private conversations for the bot now
		//if int64(update.Message.From.ID) != update.Message.Chat.ID {
		//	continue
		//}

		// Handle commands only
		if !update.Message.IsCommand() {
			continue
		}

		// Parse commands
		switch update.Message.Command() {
		case "add":
			// handle add command
			commands.AddCommand(Bot, &update)
		case "delete":
			// handle delete command
			commands.DeleteCommand(Bot, &update)
		case "list":
			// handle list command
			commands.ListCommand(Bot, &update)
		case "start":
			// handle start command
			commands.StartCommand(Bot, &update)
		case "help":
			// handle help command
			commands.HelpCommand(Bot, &update)
		}
	}
}
