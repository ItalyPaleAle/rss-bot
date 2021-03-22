package notifybot

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ItalyPaleAle/rss-bot/bot"
)

// TODO: Make env var
const listen = "127.0.0.1:8080"

// WebhookRequestPayload is the format of requests to the webhook server when using JSON
// We require the "message" key with the message to send
// Optionally, a "markdown" and "html" booleans can be set to use markdown or HTML for formatting
type WebhookRequestPayload struct {
	Message  string `json:"message"`
	Markdown bool   `json:"markdown"`
	HTML     bool   `json:"html"`
}

// NotifyBot is the class that manages the Webhook notifier
type NotifyBot struct {
	log     *log.Logger
	manager *bot.BotManager
	ctx     context.Context
	cancel  context.CancelFunc
}

// Init the object
func (nb *NotifyBot) Init(manager *bot.BotManager) error {
	// Init the logger
	nb.log = log.New(os.Stdout, "notify-bot: ", log.Ldate|log.Ltime|log.LUTC)

	// Store the manager
	nb.manager = manager

	return nil
}

// Start the web server
func (nb *NotifyBot) Start() error {
	// Context, that can be used to stop the web server (and the bot)
	nb.ctx, nb.cancel = context.WithCancel(context.Background())

	// Register all commands
	err := nb.registerRoutes()
	if err != nil {
		return err
	}

	// Create the HTTP server
	srv := &http.Server{
		Addr:           listen,
		Handler:        nb.requestHandler(),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	srv.SetKeepAlivesEnabled(false)

	// In a separate goroutine, listen for the cancelation signal to stop the server
	go func() {
		// Block until the context is canceled
		<-nb.ctx.Done()
		nb.log.Println("Shutting down the web server")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			nb.log.Fatal("Could not shut down the server gracefully", err)
			return
		}
	}()

	// Start the server in a separate goroutine so we don't block the main thread
	go func() {
		nb.log.Println("Starting the web server on, listening on", listen)
		// This call blocks until the server is shut down
		err = srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			nb.log.Fatal("Could not start the server", err)
			return
		}
	}()

	return nil
}

// Stop the background processes
func (nb *NotifyBot) Stop() {
	nb.cancel()
}

// Register all routes
func (nb *NotifyBot) registerRoutes() (err error) {
	err = nb.manager.AddRoute("(?i)^(new|add) webhook", nb.routeNew)
	if err != nil {
		return err
	}
	err = nb.manager.AddRoute("(?i)^list webhook(s?)", nb.routeList)
	if err != nil {
		return err
	}
	err = nb.manager.AddRoute("(?i)^(remove|delete) webhook", nb.routeRemove)
	if err != nil {
		return err
	}

	return nil
}
