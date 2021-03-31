package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"

	"github.com/ItalyPaleAle/rss-bot/bots/feedbot/feeds"
	"github.com/ItalyPaleAle/rss-bot/client-go"
	pb "github.com/ItalyPaleAle/rss-bot/model"
	"github.com/ItalyPaleAle/rss-bot/utils"
)

// FeedBot is the class that manages the RSS bot
type FeedBot struct {
	log    *log.Logger
	client *client.BotClient
	feeds  *feeds.Feeds
	ctx    context.Context
	cancel context.CancelFunc
}

// Init the object
func (fb *FeedBot) Init() error {
	// Init the logger
	fb.log = log.New(os.Stdout, "feedbot: ", log.Ldate|log.Ltime|log.LUTC)

	// Create a client object
	fb.client = &client.BotClient{}
	err := fb.client.Init("feeds", "Feeds", "TODO")
	if err != nil {
		return err
	}

	// Register all commands
	err = fb.registerRoutes()
	if err != nil {
		return err
	}

	return nil
}

// Start the background workers
func (fb *FeedBot) Start() error {
	// Context, that can be used to stop the bot
	fb.ctx, fb.cancel = context.WithCancel(context.Background())

	// Connect to the bot
	err := fb.client.Connect()
	if err != nil {
		return err
	}

	// TODO: BLOCK GOROUTINE UNTIL CONNECTED

	// Init the feeds object
	fb.feeds = &feeds.Feeds{}
	err = fb.feeds.Init(fb.ctx)
	if err != nil {
		return err
	}

	// Start the background worker
	go fb.backgroundWorker()
	fb.log.Println("FeedBot workers started")

	return nil
}

// Stop the background processes
func (fb *FeedBot) Stop() error {
	fb.cancel()
	err := fb.client.Disconnect()
	if err != nil {
		return err
	}
	return nil
}

// In background, start updating feeds periodically and send messages on new posts
// Also watch for the stop message
func (fb *FeedBot) backgroundWorker() {
	// Sleep for 2 seconds
	time.Sleep(2 * time.Second)

	// Channel for receiving messages to send
	msgCh := make(chan feeds.UpdateMessage)
	fb.feeds.SetUpdateChan(msgCh)

	// Queue an update right away
	fb.feeds.QueueUpdate()

	// Ticker for updates
	ticker := time.NewTicker(viper.GetDuration("FeedUpdateInterval") * time.Second)
	for {
		select {
		// On the interval, queue an update
		case <-ticker.C:
			fb.feeds.QueueUpdate()

		// Send messages on new posts
		case msg := <-msgCh:
			// This method logs errors already
			fb.sendFeedUpdate(&msg)

		// Context canceled
		case <-fb.ctx.Done():
			// Stop the ticker
			ticker.Stop()
			return
		}
	}
}

// Sends a message with a feed's post
func (fb *FeedBot) sendFeedUpdate(msg *feeds.UpdateMessage) {
	// If there's a photo, send the photo and then the message as caption
	// Note that this might fail, for example if the image is too big (>5MB)
	if msg.Post.Photo != "" {
		_, err := fb.client.SendMessage(&pb.OutMessage{
			Recipient: strconv.FormatInt(int64(msg.ChatId), 10),
			Content: &pb.OutMessage_Photo{
				Photo: &pb.OutPhotoMessage{
					File: &pb.OutFileMessage{
						Location: &pb.OutFileMessage_Url{
							Url: msg.Post.Photo,
						},
					},
					Caption:          fb.formatUpdateMessage(msg),
					CaptionParseMode: pb.ParseMode_HTML,
				},
			},
			Options: &pb.OutMessageOptions{
				DisableWebPagePreview: true,
			},
		})
		if err != nil {
			// If this failed with error "wrong file identifier/HTTP URL specified", it means that the photo filed to send, for example because it was > 5MB
			// So, just re-send the message without any photo
			if err.Error() == "telegram: wrong file identifier/HTTP URL specified (400)" {
				fb.log.Printf("Error sending photo %s to chat %d. Is the photo too big? Will re-send message without photo: %s\n", msg.Post.Photo, msg.ChatId, err.Error())
				fb.sendFeedUpdateText(msg)
			} else {
				// Just log the error and continue
				fb.log.Printf("Error sending photo %s to chat %d: %s\n", msg.Post.Photo, msg.ChatId, err.Error())
			}
		}
	} else {
		// Send the post
		fb.sendFeedUpdateText(msg)
	}
}

// Sends a message with a feed's post without photos/images
func (fb *FeedBot) sendFeedUpdateText(msg *feeds.UpdateMessage) {
	_, err := fb.client.SendMessage(&pb.OutMessage{
		Recipient: strconv.FormatInt(int64(msg.ChatId), 10),
		Content: &pb.OutMessage_Text{
			Text: &pb.OutTextMessage{
				Text:      fb.formatUpdateMessage(msg),
				ParseMode: pb.ParseMode_HTML,
			},
		},
		Options: &pb.OutMessageOptions{
			DisableWebPagePreview: true,
		},
	})
	if err != nil {
		fb.log.Printf("Error sending message to chat %d: %s\n", msg.ChatId, err.Error())
		return
	}
}

// Formats a message with an update
func (fb *FeedBot) formatUpdateMessage(msg *feeds.UpdateMessage) string {
	// Note: the msg.Feed object might be nil when passed to this method
	out := ""
	if msg.Feed != nil {
		out += fmt.Sprintf("🎙 %s:\n", utils.EscapeHTMLEntities(msg.Feed.Title))
	}

	// Add the content
	out += fmt.Sprintf("📬 <b>%s</b>\n🕓 %s\n🔗 %s\n",
		utils.EscapeHTMLEntities(msg.Post.Title),
		utils.EscapeHTMLEntities(msg.Post.Date.UTC().Format("Mon, 02 Jan 2006 15:04:05 MST")),
		utils.EscapeHTMLEntities(msg.Post.Link),
	)
	return out
}

// Register all routes
func (fb *FeedBot) registerRoutes() (err error) {
	err = fb.client.AddRoute("(?i)^add feed", func(m *pb.InMessage) error {
		// Errors are already handled by the method
		fb.routeAdd(m)
		return nil
	})
	if err != nil {
		return err
	}
	err = fb.client.AddRoute("(?i)^list feed(s?)", func(m *pb.InMessage) error {
		// Errors are already handled by the method
		fb.routeList(m)
		return nil
	})
	if err != nil {
		return err
	}
	err = fb.client.AddRoute("(?i)^(remove|delete) feed", func(m *pb.InMessage) error {
		// Errors are already handled by the method
		fb.routeRemove(m)
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
