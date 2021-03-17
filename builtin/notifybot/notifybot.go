package notifybot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ItalyPaleAle/rss-bot/bot"
)

// TODO: Make env var
const listen = ":8080"

// Maximum request body size is 4 KB
const maxBodySize = int64(4 << 10)

// WebhookRequestPayload is the format of requests to the webhook server: we support sending messages in the "message" key only
type WebhookRequestPayload struct {
	Message string `json:"message"`
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

	// Start the server
	nb.log.Println("Starting the web server on, listening on", listen)
	// This call blocks until the server is shut down
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Stop the background processes
func (fb *NotifyBot) Stop() {
	fb.cancel()
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

// Handler for the requests received by the server
func (fb *NotifyBot) requestHandler() http.Handler {
	// The path must start with /webhook and then contain the recipient ID
	reqUrlExpr := regexp.MustCompile("^/webhook/([A-Za-z0-9_-]+)$")

	// Matching "Bearer" at the beginning of the access token in the Authorization header
	bearerExpr := regexp.MustCompile("((?i)^Bearer )?([A-Za-z0-9_-]+)$")

	// Specs: https://github.com/cloudevents/spec/blob/v1.0.1/http-webhook.md
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		// Must only respond to POST requests
		if r.Method != "POST" {
			responseError(w, "Not found", http.StatusNotFound)
			return
		}

		// Match the URL
		match := reqUrlExpr.FindStringSubmatch(r.URL.Path)
		if len(match) < 2 || match[1] == "" {
			responseError(w, "Not found", http.StatusNotFound)
			return
		}

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

		// Validate the auth header
		// TODO: Use the database to validate tokens. Note: if the recipient ID doesn't exist, return a 401 too just like if the auth token is wrong, to avoid giving away information about recipients
		if auth == "" {
			responseError(w, "Invalid authorization", http.StatusUnauthorized)
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
			fb.log.Println("Error while reading request body", err)
			responseError(w, "The request body could not be read", http.StatusUnsupportedMediaType)
			return
		}
		payload := &WebhookRequestPayload{}
		err = json.Unmarshal(body, payload)
		if err != nil {
			fb.log.Println("Error while parsing request body", err)
			responseError(w, "The request body could not be parsed as JSON", http.StatusUnsupportedMediaType)
			return
		}

		// Ensure the message key was present in the request payload
		if payload.Message == "" {
			responseError(w, "The key 'message' was missing in the request", http.StatusUnsupportedMediaType)
			return
		}

		fmt.Println(payload.Message, auth)

		// Respond with an "Accepted" (202) status code
		w.WriteHeader(http.StatusAccepted)
	})
}

// Register all routes
func (fb *NotifyBot) registerRoutes() (err error) {
	/*fb.manager.AddRoute("(?i)^add feed", fb.routeAdd)
	fb.manager.AddRoute("(?i)^list feed(s?)", fb.routeList)
	fb.manager.AddRoute("(?i)^remove feed", fb.routeRemove)*/

	return nil
}
