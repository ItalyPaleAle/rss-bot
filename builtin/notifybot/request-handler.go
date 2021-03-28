package notifybot

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/ItalyPaleAle/rss-bot/db"
	pb "github.com/ItalyPaleAle/rss-bot/service"
)

// Maximum request body size is 4 KB
const maxBodySize = int64(4 << 10)

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

		// Validate the authorization for this webhook recipient and get the chat ID
		chatId, authOk := validateAuth(w, r, recipientId)
		if !authOk || chatId == 0 {
			// Response was already sent
			return
		}

		// Limit reading to the maximum request body
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		// Handle request depending on Content-Type
		ct := r.Header.Get("content-type")
		parseMode := pb.ParseMode_PLAIN
		var message string
		switch {
		// Plain-text request
		case strings.HasPrefix(ct, "text/plain"):
			// Read the entire body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				nb.log.Println("Error while reading request body", err)
				responseError(w, "The request body could not be read", http.StatusUnsupportedMediaType)
				return
			}
			if len(body) == 0 || int64(len(body)) > maxBodySize {
				nb.log.Println("Invalid body length", len(body))
				responseError(w, "The request body is empty", http.StatusUnsupportedMediaType)
				return
			}
			message = string(body)

		// JSON request
		case strings.HasPrefix(ct, "application/json"):
			// Parse the content as JSON
			decoder := json.NewDecoder(r.Body)
			payload := &WebhookRequestPayload{}
			err := decoder.Decode(payload)
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
			message = payload.Message

			// Markdown or HTML formatting
			if payload.HTML {
				parseMode = pb.ParseMode_HTML
			} else if payload.Markdown {
				parseMode = pb.ParseMode_MARKDOWN_V2
			}

		// We only accept requests with data in JSON or plain text for now
		default:
			responseError(w, "This webhook accepts requests in plain text (text/plain) or JSON (application/json) formats only", http.StatusUnsupportedMediaType)
			return
		}

		// Append the recipient ID to the message
		switch parseMode {
		case pb.ParseMode_HTML:
			message += "\n<i>(" + recipientId + ")</i>"
		case pb.ParseMode_MARKDOWN_V2:
			message += "\n*(" + recipientId + ")*"
		default:
			message += "\n(" + recipientId + ")"
		}

		// Send the message in a background goroutine (so we're not pausing the response)
		go func() {
			_, err := nb.manager.SendMessage(&pb.OutMessage{
				Recipient: strconv.FormatInt(chatId, 10),
				Content: &pb.OutMessage_Text{
					Text: &pb.OutTextMessage{
						Text:      message,
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
		if len(match) == 3 && len(match[2]) != 0 {
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
