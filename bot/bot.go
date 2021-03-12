package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/ItalyPaleAle/rss-bot/feeds"
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
	authKey := viper.GetString("TelegramAuthToken")
	if authKey == "" || authKey == "token" {
		return errors.New("Telegram auth key not set. Please make sure that the 'TelegramAuthToken' option is present in the config file, or use the 'BOT_TELEGRAMAUTHTOKEN' environmental variable.")
	}

	// Poller
	var poller tb.Poller = &tb.LongPoller{Timeout: 10 * time.Second}

	// Check if we're restricting the bot to certain users only
	allowedUsers := b.getAllowedUsers()
	if len(allowedUsers) > 0 {
		// Create a middleware
		poller = tb.NewMiddlewarePoller(poller, func(u *tb.Update) bool {
			if u.Message == nil {
				return true
			}

			// Restrict to certain users only
			if u.Message.Sender == nil || u.Message.Sender.ID == 0 || !allowedUsers[u.Message.Sender.ID] {
				if u.Message.Sender == nil {
					b.log.Println("Ignoring message from empty sender")
				} else {
					b.log.Println("Ignoring message from un-allowed sender:", u.Message.Sender.ID)
				}
				return false
			}

			return true
		})
	}

	// Create the bot object
	// TODO: Enable support for webhook: https://godoc.org/gopkg.in/tucnak/telebot.v2#Webhook
	b.bot, err = tb.NewBot(tb.Settings{
		Token:   authKey,
		Poller:  poller,
		Verbose: viper.GetBool("TelegramAPIDebug"),
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
	ticker := time.NewTicker(viper.GetDuration("FeedUpdateInterval") * time.Second)
	for {
		select {
		// On the interval, queue an update
		case <-ticker.C:
			b.feeds.QueueUpdate()

		// Send messages on new posts
		case msg := <-msgCh:
			// This method logs errors already
			b.sendFeedUpdate(tb.ChatID(msg.ChatId), &msg)

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

// Returns a string in which HTML entities are escaped as required by Telegram: <>&
func (b *RSSBot) escapeHTMLEntities(s string) string {
	r := strings.NewReplacer("<", "&lt;", ">", "&gt;", "&", "&amp;")
	return r.Replace(s)
}

// Sends a message with a feed's post
func (b *RSSBot) sendFeedUpdate(recipient tb.Recipient, msg *feeds.UpdateMessage) {
	// Send title
	_, err := b.bot.Send(
		recipient,
		b.formatUpdateMessage(msg),
		&tb.SendOptions{
			ParseMode:             tb.ModeHTML,
			DisableWebPagePreview: true,
		},
	)
	if err != nil {
		b.log.Printf("Error sending message to chat %d: %s\n", msg.ChatId, err.Error())
		return
	}

	// Send photo, if any
	// Note that this might fail, for example if the image is too big (>5MB)
	if msg.Post.Photo != "" {
		_, err = b.bot.Send(
			recipient,
			&tb.Photo{File: tb.FromURL(msg.Post.Photo)},
			&tb.SendOptions{
				// Do not send notifications for subsequent messages
				DisableNotification: true,
			},
		)
		if err != nil {
			b.log.Printf("Error sending photo %s to chat %d: %s\n", msg.Post.Photo, msg.ChatId, err.Error())
		}
	}
}

// Formats a message with an update
func (b *RSSBot) formatUpdateMessage(msg *feeds.UpdateMessage) string {
	// Note: the msg.Feed object might be nil when passed to this method
	out := ""
	if msg.Feed != nil {
		out += fmt.Sprintf("🎙 %s:\n", b.escapeHTMLEntities(msg.Feed.Title))
	}

	// Add the content
	out += fmt.Sprintf("📬 <b>%s</b>\n🕓 %s\n🔗 %s\n",
		b.escapeHTMLEntities(msg.Post.Title),
		b.escapeHTMLEntities(msg.Post.Date.UTC().Format("Mon, 02 Jan 2006 15:04:05 MST")),
		b.escapeHTMLEntities(msg.Post.Link),
	)
	return out
}

// Sends a response to a command
// For commands sent in private chats, this just sends a regular message
// In groups, this replies to a specific message
func (b *RSSBot) respondToCommand(m *tb.Message, msg interface{}, options ...interface{}) (out *tb.Message, err error) {
	// If it's a private chat, send a message, otherwise reply
	if m.Private() {
		out, err = b.bot.Send(m.Sender, msg, options...)
	} else {
		out, err = b.bot.Reply(m, msg, options...)
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
	b.bot.Handle("/remove", b.handleRemove)

	// Handler for callbacks
	b.bot.Handle(tb.OnCallback, func(cb *tb.Callback) {
		// Seems that we need to trim whitespaces from the data
		data := strings.TrimSpace(cb.Data)
		// The main command comes before the /
		pos := strings.Index(data, "/")
		cmd := data
		var userData string
		if pos > -1 {
			cmd = data[0:pos]
			userData = data[(pos + 1):]
		}

		switch cmd {
		// Cancel command removes all inline keyboards
		case "cancel":
			_, err := b.bot.Edit(cb.Message, "Ok, I won't do anything")
			if err != nil {
				b.log.Printf("Error canceling callback: %s\n", err.Error())
			}

		// Confirm removing a feed
		case "confirm-remove":
			b.callbackConfirmRemove(cb, userData)
		}
	})

	// Set commands for Telegram
	err = b.bot.SetCommands([]tb.Command{
		{Text: "add", Description: "Subscribe to a new feed"},
		{Text: "list", Description: "List subscriptions for this chat"},
		{Text: "remove", Description: "Unsubscribe from a feed"},
		{Text: "help", Description: "Show help message"},
	})
	return err
}

// Returns the list of allowed users (if any)
// Returns a map so lookups are faster
func (b *RSSBot) getAllowedUsers() (allowedUsers map[int]bool) {
	// Check if we can get an int slice
	uids := viper.GetIntSlice("AllowedUsers")
	if len(uids) == 0 {
		// Check if we can get a string
		str := viper.GetString("AllowedUsers")
		if str != "" {
			// Split on commas
			for _, s := range strings.Split(str, ",") {
				// Ignore invalid ones
				num, err := strconv.Atoi(s)
				if err != nil || num < 1 {
					continue
				}
				// Add to the map
				if allowedUsers == nil {
					allowedUsers = make(map[int]bool)
				}
				allowedUsers[num] = true
			}
		}
	} else {
		// Convert to a map
		allowedUsers = make(map[int]bool, len(uids))
		for i := 0; i < len(uids); i++ {
			allowedUsers[uids[i]] = true
		}
	}
	return
}
