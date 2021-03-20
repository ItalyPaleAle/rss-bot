package notifybot

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ItalyPaleAle/rss-bot/bot"
	"github.com/ItalyPaleAle/rss-bot/db"
	pb "github.com/ItalyPaleAle/rss-bot/proto"
)

// TODO: Make env var
const listen = "127.0.0.1:8080"

// Maximum request body size is 4 KB
const maxBodySize = int64(4 << 10)

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

// Handler for the requests received by the server
func (nb *NotifyBot) requestHandler() http.Handler {
	// The path must start with /webhook and then contain the recipient ID
	reqUrlExpr := regexp.MustCompile("^/webhook/([A-Za-z0-9_-]+)$")

	// Specs: https://github.com/cloudevents/spec/blob/v1.0.1/http-webhook.md
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		// Must only respond to POST requests
		if r.Method != "POST" {
			responseError(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// Match the URL
		match := reqUrlExpr.FindStringSubmatch(r.URL.Path)
		if len(match) < 2 || match[1] == "" {
			responseError(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		recipientId := match[1]

		// Get the authorization token, from the "Authorization" header first and then the querystring
		auth := ""
		if a := r.Header.Get("authorization"); a != "" {
			// Match with the "Bearer" optional prefix
			match := bearerExpr.FindStringSubmatch(a)
			if len(match) == 2 && len(match[2]) != 0 {
				auth = match[2]
			}
		}
		if auth == "" {
			query := r.URL.Query()
			if query != nil {
				if a := query.Get("access_token"); a != "" {
					auth = a
				}
			}
		}

		// Validate the authorization for this webhook recipient and get the chat ID
		chatId, authOk := validateAuth(w, r, recipientId)
		if !authOk || chatId == 0 {
			// Response was already sent
			return
		}

		// We only accept requests with data in JSON for now
		if !strings.HasPrefix(r.Header.Get("content-type"), "application/json") {
			responseError(w, "This webhook accepts requests in JSON format only", http.StatusUnsupportedMediaType)
			return
		}

		// Limit reading to the maximum request body
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		// Parse the content as JSON
		body, err := io.ReadAll(r.Body)
		if err != nil {
			nb.log.Println("Error while reading request body", err)
			responseError(w, "The request body could not be read", http.StatusUnsupportedMediaType)
			return
		}
		payload := &WebhookRequestPayload{}
		err = json.Unmarshal(body, payload)
		if err != nil {
			nb.log.Println("Error while parsing request body", err)
			responseError(w, "The request body could not be parsed as JSON", http.StatusUnsupportedMediaType)
			return
		}

		// Ensure the message key was present in the request payload
		if payload.Message == "" {
			responseError(w, "The key 'message' was missing in the request", http.StatusUnsupportedMediaType)
			return
		}

		// Send the message in a background goroutine (so we're not pausing the response)
		go func() {
			// Markdown or HTML formatting
			parseMode := pb.ParseMode_PLAIN
			if payload.HTML {
				parseMode = pb.ParseMode_HTML
			} else if payload.Markdown {
				parseMode = pb.ParseMode_MARKDOWN_V2
			}

			_, err := nb.manager.SendMessage(&pb.OutMessage{
				Recipient: strconv.FormatInt(chatId, 10),
				Content: &pb.OutMessage_Text{
					Text: &pb.OutTextMessage{
						Text:      payload.Message,
						ParseMode: parseMode,
					},
				},
			})
			if err != nil {
				nb.log.Println("Error while sending the notification", err)
			}
		}()

		// Respond with an "Accepted" (202) status code
		w.WriteHeader(http.StatusAccepted)
	})
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

	return nil
}

// Internal function used to return a HTTP error (formatted as JSON) in the response
func responseError(w http.ResponseWriter, errMsg string, statusCode int) {
	// Headers and status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Write the response as JSON
	j := json.NewEncoder(w)
	_ = j.Encode(struct {
		Error string `json:"error"`
	}{
		Error: errMsg,
	})
}

// Matching "Bearer" at the beginning of the access token in the Authorization header
var bearerExpr = regexp.MustCompile("((?i)^Bearer )?([A-Za-z0-9_-]+)$")

// Internal function used to validate authorization to use the webhook
func validateAuth(w http.ResponseWriter, r *http.Request, recipientId string) (chatId int64, ok bool) {
	// Get the auth token, from the "Authorization" header first and then the querystring
	auth := ""
	if a := r.Header.Get("authorization"); a != "" {
		// Match with the "Bearer" optional prefix
		match := bearerExpr.FindStringSubmatch(a)
		if len(match) == 2 && len(match[2]) != 0 {
			auth = match[2]
		}
	}
	if auth == "" {
		query := r.URL.Query()
		if query != nil {
			if a := query.Get("access_token"); a != "" {
				auth = a
			}
		}
	}

	// Validate the auth token
	DB := db.GetDB()
	if auth == "" {
		responseError(w, "Invalid authorization", http.StatusUnauthorized)
		return 0, false
	}
	authHash := sha256.Sum256([]byte(auth))
	webhook := &db.Webhook{}
	err := DB.Get(webhook, "SELECT * FROM webhooks WHERE webhook_id = ? AND webhook_key = ?", recipientId, authHash[:])
	if err == sql.ErrNoRows || webhook.ChatID == 0 {
		responseError(w, "Invalid authorization", http.StatusUnauthorized)
		return 0, false
	} else if err != nil {
		responseError(w, "Internal error", http.StatusInternalServerError)
		return 0, false
	}

	return webhook.ChatID, true
}
