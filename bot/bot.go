package bot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"google.golang.org/protobuf/types/known/timestamppb"
	tb "gopkg.in/tucnak/telebot.v2"

	pb "github.com/ItalyPaleAle/rss-bot/proto"
)

// BotManager is the class that manages the bot
type BotManager struct {
	log    *log.Logger
	bot    *tb.Bot
	ctx    context.Context
	cancel context.CancelFunc
	routes []routeDefinition
}

// Init the object
func (b *BotManager) Init() (err error) {
	// Init the logger
	b.log = log.New(os.Stdout, "bot: ", log.Ldate|log.Ltime|log.LUTC)

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
		poller = tb.NewMiddlewarePoller(poller, b.allowedUsersMiddleware(allowedUsers))
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

	// Handle messages
	b.handleMessages()

	return nil
}

// Start the background workers
func (b *BotManager) Start() error {
	// Context, that can be used to stop the bot
	b.ctx, b.cancel = context.WithCancel(context.Background())

	// Start the bot
	b.log.Println("Bot starting")
	b.bot.Start()

	return nil
}

// Stop the bot and the background processes
func (b *BotManager) Stop() {
	b.cancel()
}

// SendMessage sends a message to a chat or user
func (b *BotManager) SendMessage(msg *pb.OutMessage) (*pb.SentMessage, error) {
	// Ensure we have a recipient
	if msg.Recipient == "" {
		return nil, errors.New("Message does not have any recipient")
	}

	// Convert the recipient to an object that implements tb.Recipient
	recipient := msgRecipient{msg.Recipient}

	// Content
	var content interface{}

	// Send options
	opts := &tb.SendOptions{}
	if msg.DisableNotification {
		opts.DisableNotification = true
	}
	if msg.DisableWebPagePreview {
		opts.DisableWebPagePreview = true
	}

	// Check if we're replying to a message
	if msg.ReplyTo > 0 {
		// Check if we can safely cast from int64 to int
		if int64(int(msg.ReplyTo)) != msg.ReplyTo {
			return nil, errors.New("Conversion of message ID to reply to would overflow")
		}
		opts.ReplyTo = &tb.Message{ID: int(msg.ReplyTo)}
	}

	// Process the message depending on its type
	switch c := msg.Content.(type) {
	case *pb.OutMessage_Text:
		// Text message
		if c.Text == nil || c.Text.Text == "" {
			return nil, errors.New("Message's text content is empty")
		}
		content = c.Text.Text

		// Set parse mode, if needed
		switch c.Text.ParseMode {
		case pb.ParseMode_HTML:
			opts.ParseMode = tb.ModeHTML
		case pb.ParseMode_MARKDOWN:
			opts.ParseMode = tb.ModeMarkdown
		case pb.ParseMode_MARKDOWN_V2:
			opts.ParseMode = tb.ModeMarkdownV2
		}

	case *pb.OutMessage_File:
		// Message is a file
		if c.File == nil || c.File.Location == nil {
			return nil, errors.New("Message's file location is empty or invalid")
		}
		switch f := c.File.Location.(type) {
		case *pb.OutFileMessage_Url:
			if f.Url == "" {
				return nil, errors.New("Message's file URL is empty or invalid")
			}
			content = tb.FromURL(f.Url)
		case *pb.OutFileMessage_LocalPath:
			if f.LocalPath == "" {
				return nil, errors.New("Message's file local path is empty or invalid")
			}
			content = tb.FromDisk(f.LocalPath)
		case *pb.OutFileMessage_Data:
			if len(f.Data) == 0 {
				return nil, errors.New("Message's file data is empty or invalid")
			}
			if len(f.Data) > 20*1024*1024 {
				return nil, errors.New("Message's file data is too long - maximum size is 20MB")
			}
			content = tb.FromReader(bytes.NewReader(f.Data))
		default:
			return nil, errors.New("Message's file location is empty or invalid")
		}

	case *pb.OutMessage_Photo:
		// Message is a photo
		if c.Photo == nil || c.Photo.File == nil || c.Photo.File.Location == nil {
			return nil, errors.New("Message's photo location is empty or invalid")
		}
		switch f := c.Photo.File.Location.(type) {
		case *pb.OutFileMessage_Url:
			if f.Url == "" {
				return nil, errors.New("Message's photo URL is empty or invalid")
			}
			content = &tb.Photo{File: tb.FromURL(f.Url)}
		case *pb.OutFileMessage_LocalPath:
			if f.LocalPath == "" {
				return nil, errors.New("Message's photo local path is empty or invalid")
			}
			content = &tb.Photo{File: tb.FromDisk(f.LocalPath)}
		case *pb.OutFileMessage_Data:
			if len(f.Data) == 0 {
				return nil, errors.New("Message's photo data is empty or invalid")
			}
			if len(f.Data) > 20*1024*1024 {
				return nil, errors.New("Message's photo data is too long - maximum size is 20MB")
			}
			content = &tb.Photo{File: tb.FromReader(bytes.NewReader(f.Data))}
		default:
			return nil, errors.New("Message's photo location is empty or invalid")
		}
	default:
		// Message's type is empty or invalid, so return
		return nil, errors.New("Message's content is empty or invalid")
	}

	// Send the message
	sent, err := b.bot.Send(recipient, content, opts)
	if err != nil {
		return nil, err
	}

	// Get the ID of the message that was sent
	res := &pb.SentMessage{
		MessageId: int64(sent.ID),
		ChatId:    sent.Chat.ID,
	}

	return res, nil
}

// EditTextMessage requests an edit to a text message that was sent before
func (b *BotManager) EditTextMessage(edit *pb.EditTextMessage) error {
	// Message signature
	msg := msgEditable{
		MessageId: strconv.FormatInt(edit.Message.MessageId, 10),
		ChatId:    edit.Message.ChatId,
	}

	// Send options
	opts := &tb.SendOptions{}
	if edit.DisableWebPagePreview {
		opts.DisableWebPagePreview = true
	}

	// Content
	if edit.Text == nil || edit.Text.Text == "" {
		return errors.New("Message's text content is empty")
	}
	content := edit.Text.Text

	// Set parse mode, if needed
	switch edit.Text.ParseMode {
	case pb.ParseMode_HTML:
		opts.ParseMode = tb.ModeHTML
	case pb.ParseMode_MARKDOWN:
		opts.ParseMode = tb.ModeMarkdown
	case pb.ParseMode_MARKDOWN_V2:
		opts.ParseMode = tb.ModeMarkdownV2
	}

	// Request the edit
	_, err := b.bot.Edit(msg, content, opts)
	if err != nil {
		return err
	}
	return nil
}

// AddRoute adds a route for text messages
func (b *BotManager) AddRoute(route string, cb RouteCallback) error {
	if len(route) < 1 {
		return errors.New("Route is empty or invalid")
	}
	if cb == nil {
		return errors.New("Callback is empty")
	}

	// Create a regular expression from the route
	exp, err := regexp.Compile(route)
	if err != nil {
		return fmt.Errorf("Could not compile route's regular expression: %s", err)
	}

	// Add the route to the list
	b.routes = append(b.routes, routeDefinition{
		Path:     route,
		Match:    exp,
		Callback: cb,
	})

	return nil
}

// Adds the core routes
func (b *BotManager) addCoreRoutes() {
	b.routes = []routeDefinition{
		// Say hi!
		{
			Match: regexp.MustCompile("(?i)^(hi|hello|hey)([[:punct:]]|\\s)*(there|bot)?"),
			Callback: func(mp *pb.InMessage) {
				_, err := b.RespondToCommand(mp, "👋 Hey there! What can I do for you? ")
				if err != nil {
					// Log errors only
					b.log.Printf("Error sending message to chat %d: %s\n", mp.ChatId, err.Error())
				}
			},
		},
		// Add a route for help messages
		{
			Match: regexp.MustCompile("(?i)^help"),
			Callback: func(m *pb.InMessage) {
				b.helpMessageCallback(m)
			},
		},
	}
}

// Sends the help message
func (b *BotManager) helpMessageCallback(m *pb.InMessage) {
	b.RespondToCommand(m, "Here's where I'll write the help message 🤔")
}

// Finds a route matching the message, if any
func (b *BotManager) matchRoute(m *tb.Message) RouteCallback {
	// Iterate through all routes until we find a matching one
	for _, r := range b.routes {
		if r.Match.MatchString(m.Text) {
			return r.Callback
		}
	}

	return nil
}

// RespondToCommand sends a response to a command
// For commands sent in private chats, this just sends a regular message
// In groups, this replies to a specific message
func (b *BotManager) RespondToCommand(in *pb.InMessage, content interface{}) (*pb.SentMessage, error) {
	// Message to send
	out := &pb.OutMessage{
		Recipient: strconv.FormatInt(in.ChatId, 10),
	}

	// Content
	switch c := content.(type) {
	case *pb.OutMessage_Text:
	case *pb.OutMessage_File:
	case *pb.OutMessage_Photo:
		// Already in the right format
		out.Content = c
	case *pb.OutTextMessage:
		// Text message
		out.Content = &pb.OutMessage_Text{
			Text: c,
		}
	case *pb.OutFileMessage:
		// File
		out.Content = &pb.OutMessage_File{
			File: c,
		}
	case *pb.OutPhotoMessage:
		// Photo
		out.Content = &pb.OutMessage_Photo{
			Photo: c,
		}
	case string:
		// String
		out.Content = &pb.OutMessage_Text{
			Text: &pb.OutTextMessage{
				Text: c,
			},
		}
	default:
		return nil, errors.New("Invalid content argument")
	}

	// If it's a private chat, send as a regular message, otherwise reply
	out.ReplyTo = 0
	if !in.Private {
		out.ReplyTo = in.MessageId
	}

	// Send the message
	return b.SendMessage(out)
}

// Registers the functions that handle all messages
func (b *BotManager) handleMessages() {
	// Handle the /start message
	b.bot.Handle("/start", func(m *tb.Message) {
		mp := messageToProto(m)
		_, err := b.RespondToCommand(mp, "👋 Nice to meet you!")
		if err != nil {
			// Log errors only
			b.log.Printf("Error sending message to chat %d: %s\n", mp.ChatId, err.Error())
		}
		b.helpMessageCallback(mp)
	})

	// Handle the /help message
	b.bot.Handle("/help", func(m *tb.Message) {
		mp := messageToProto(m)
		b.helpMessageCallback(mp)
	})

	// Add core routes for text messages
	b.addCoreRoutes()

	// Handle text messages that weren't captured by other handlers
	b.bot.Handle(tb.OnText, func(m *tb.Message) {
		// Trim whitespaces
		m.Text = strings.TrimSpace(m.Text)

		// Convert to the proto model
		mp := messageToProto(m)

		// Look for a matching route
		cb := b.matchRoute(m)
		if cb != nil {
			cb(mp)
			return
		}

		// Explain you didn't get that
		_, err := b.RespondToCommand(mp, "Sorry, I didn't quite get that 😔 Ask me \"help\" if you need directions.")
		if err != nil {
			// Log errors only
			b.log.Printf("Error sending message to chat %d: %s\n", mp.ChatId, err.Error())
		}
	})
}

// Returns the list of allowed users (if any)
// Returns a map so lookups are faster
func (b *BotManager) getAllowedUsers() (allowedUsers map[int]bool) {
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

// Returns the poller middleware that only allows messages from users in the allowlist
func (b *BotManager) allowedUsersMiddleware(list map[int]bool) func(u *tb.Update) bool {
	return func(u *tb.Update) bool {
		if u.Message == nil {
			return true
		}

		// Restrict to certain users only
		if u.Message.Sender == nil || u.Message.Sender.ID == 0 || !list[u.Message.Sender.ID] {
			if u.Message.Sender == nil {
				b.log.Println("Ignoring message from empty sender")
			} else {
				b.log.Println("Ignoring message from disallowed sender:", u.Message.Sender.ID)
			}
			return false
		}

		return true
	}
}

// Implements the tb.Recipient interface
type msgRecipient struct {
	R string
}

// Recipient returns the recipient of the message
func (m msgRecipient) Recipient() string {
	return m.R
}

// Returns a msgRecipient object from a chatId
func recipientFromChatId(chatID int64) msgRecipient {
	return msgRecipient{strconv.FormatInt(chatID, 10)}
}

// Implements the tb.Editable interface
type msgEditable struct {
	MessageId string
	ChatId    int64
}

// MessageSig returns the message signature
func (m msgEditable) MessageSig() (messageID string, chatID int64) {
	return m.MessageId, m.ChatId
}

// Converts a message from telebot (tb.Message) into the protobuf model
func messageToProto(m *tb.Message) *pb.InMessage {
	return &pb.InMessage{
		MessageId: int64(m.ID),
		SenderId:  int64(m.Sender.ID),
		ChatId:    m.Chat.ID,
		Time:      timestamppb.New(m.Time()),
		Private:   m.Private(),
		Text:      m.Text,
	}
}

// RouteCallback is the callback function for a given route
type RouteCallback func(m *pb.InMessage)

// Internal struct used to maintain a route definition
type routeDefinition struct {
	Path     string
	Match    *regexp.Regexp
	Callback RouteCallback
}
