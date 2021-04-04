package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ItalyPaleAle/rss-bot/client-go"
	pb "github.com/ItalyPaleAle/rss-bot/model"
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
	log    *log.Logger
	client *client.BotClient
	ctx    context.Context
	cancel context.CancelFunc
}

// Init the object
func (nb *NotifyBot) Init() error {
	// Init the logger
	nb.log = log.New(os.Stdout, "notifybot: ", log.Ldate|log.Ltime|log.LUTC)

	// Create a client object
	nb.client = &client.BotClient{}
	err := nb.client.Init("notify", "Notifier", "TODO")
	if err != nil {
		return err
	}

	// Register all commands
	err = nb.registerRoutes()
	if err != nil {
		return err
	}

	return nil
}

// Start the bot and the web server
func (nb *NotifyBot) Start() error {
	// Context, that can be used to stop the web server (and the bot)
	nb.ctx, nb.cancel = context.WithCancel(context.Background())

	// Start the bot
	err := nb.client.Start()
	if err != nil {
		return err
	}

	// TODO: BLOCK GOROUTINE UNTIL CONNECTED

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
		nb.log.Println("Starting the web server, listening on", listen)
		// This call blocks until the server is shut down
		err = srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			nb.log.Fatal("Could not start the server", err)
			return
		}
	}()

	return nil
}

// Stop the bot and the web server
func (nb *NotifyBot) Stop() error {
	nb.cancel()
	err := nb.client.Stop()
	if err != nil {
		return err
	}
	return nil
}

// Register all routes
func (nb *NotifyBot) registerRoutes() (err error) {
	err = nb.client.AddRoute("(?i)^(new|add) webhook", func(m *pb.InMessage) error {
		// Errors are already handled by the method
		nb.routeNew(m)
		return nil
	})
	if err != nil {
		return err
	}
	err = nb.client.AddRoute("(?i)^list webhook(s?)", func(m *pb.InMessage) error {
		// Errors are already handled by the method
		nb.routeList(m)
		return nil
	})
	if err != nil {
		return err
	}
	err = nb.client.AddRoute("(?i)^(remove|delete) webhook", func(m *pb.InMessage) error {
		// Errors are already handled by the method
		nb.routeRemove(m)
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
