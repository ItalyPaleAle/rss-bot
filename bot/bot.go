package bot

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/0x111/telegram-rss-bot/feeds"
)

// RSSBot is the class that manages the RSS bot
type RSSBot struct {
	log    *log.Logger
	bot    *tb.Bot
	feeds  *feeds.Feeds
	ctx    context.Context
	cancel context.CancelFunc
}

// Init the object
func (b *RSSBot) Init() (err error) {
	// Init the logger
	b.log = log.New(os.Stdout, "rss-bot: ", log.Ldate|log.Ltime|log.LUTC)

	// Get the auth key
	// "token" is the default value in the config file
	authKey := viper.GetString("telegram_auth_key")
	if authKey == "" || authKey == "token" {
		return errors.New("Telegram auth key not set")
	}

	// Create the bot object
	// TODO: Enable support for webhook: https://godoc.org/gopkg.in/tucnak/telebot.v2#Webhook
	b.bot, err = tb.NewBot(tb.Settings{
		Token:   authKey,
		Poller:  &tb.LongPoller{Timeout: 10 * time.Second},
		Verbose: viper.GetBool("telegram_api_debug"),
	})
	if err != nil {
		return err
	}

	return nil
}

// Start the background workers
func (b *RSSBot) Start() error {
	// Context, that can be used to stop the bot
	b.ctx, b.cancel = context.WithCancel(context.Background())

	// Init the feeds object
	b.feeds = &feeds.Feeds{}
	err := b.feeds.Init(b.ctx)
	if err != nil {
		return err
	}

	// Register the command handlers
	err = b.registerCommands()
	if err != nil {
		return err
	}

	// Start the background worker
	go b.backgroundWorker()

	// Start the bot
	log.Println("Bot starting")
	b.bot.Start()

	return nil
}

// Stop the bot and the background processes
func (b *RSSBot) Stop() {
	b.cancel()
}

// In background, start updating feeds periodically and send messages on new posts
// Also watch for the stop message
func (b *RSSBot) backgroundWorker() {
	// Sleep for 2 seconds
	time.Sleep(2 * time.Second)

	// Channel for receiving messages to send
	msgCh := make(chan feeds.UpdateMessage)
	b.feeds.SetUpdateChan(msgCh)

	// Queue an update right away
	b.feeds.QueueUpdate()

	// Ticker for updates
	ticker := time.NewTicker(viper.GetDuration("feed_updates_interval") * time.Second)
	for {
		select {
		// On the interval, queue an update
		case <-ticker.C:
			b.feeds.QueueUpdate()

		// Send messages on new posts
		case msg := <-msgCh:
			b.bot.Send(tb.ChatID(msg.ChatId), b.formatUpateMessage(&msg))

		// Context canceled
		case <-b.ctx.Done():
			// Stop the bot
			b.bot.Stop()
			// Stop the ticker
			ticker.Stop()
			return
		}
	}
}

// Formats a message with an update
func (b *RSSBot) formatUpateMessage(msg *feeds.UpdateMessage) string {
	// Note: the msg.Feed object might be nil when passed to this method

	// Return the link only for now
	return msg.Post.Link
}

// Sends a response to a command
// For commands sent in private chats, this just sends a regular message
// In groups, this replies to a specific message
func (b *RSSBot) respondToCommand(m *tb.Message, msg string) (out *tb.Message, err error) {
	// If it's a private chat, send a message, otherwise reply
	if m.Private() {
		out, err = b.bot.Send(m.Sender, msg)
	} else {
		out, err = b.bot.Reply(m, msg)
	}

	// Log errors
	if err != nil {
		b.log.Printf("Error sending message to chat %d: %s\n", m.Chat.ID, err.Error())
	}

	return
}

// Register all command handlers
func (b *RSSBot) registerCommands() (err error) {
	// Register handlers
	b.bot.Handle("/start", b.handleStart)
	b.bot.Handle("/help", b.handleHelp)
	b.bot.Handle("/add", b.handleAdd)
	b.bot.Handle("/list", b.handleList)

	// Set commands for Telegram
	err = b.bot.SetCommands([]tb.Command{
		{Text: "add", Description: "Subscribe to a new feed"},
		{Text: "list", Description: "List subscriptions for this chat"},
		{Text: "help", Description: "Show help message"},
	})
	return err
}
