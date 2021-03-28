package notifybot

import (
	"regexp"
	"strconv"

	"github.com/ItalyPaleAle/rss-bot/db"
	pb "github.com/ItalyPaleAle/rss-bot/service"
)

var routeRemoveMatch = regexp.MustCompile("(?i)^(remove|delete) webhook ([\\S]+)")

// Route for the "remove webhook" command
func (nb *NotifyBot) routeRemove(m *pb.InMessage) {
	// Get the arg
	match := routeRemoveMatch.FindStringSubmatch(m.Text)
	if len(match) < 3 || match[2] == "" {
		_, err := nb.manager.RespondToCommand(m, "I didn't understand this \"remove webhook\" message - is the webhook number or ID missing?")
		if err != nil {
			// Log errors only
			nb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		}
		return
	}

	// Check if we have an ID or the number of the webhook from the list
	// Assume that if the input is longer than 10, that's an ID
	queryStr := ""
	queryArgs := []interface{}{m.ChatId}
	if len(match[2]) < 10 {
		num, e := strconv.ParseInt(match[2], 10, 64)
		if e != nil || num < 1 {
			_, err := nb.manager.RespondToCommand(m, "I didn't understand this \"remove webhook\" message - is the webhook number or ID missing?")
			if err != nil {
				// Log errors only
				nb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
			}
			return
		}

		// The numeric ID is always incremented by 1 (so it starts from 1)
		queryStr = "DELETE FROM webhooks WHERE ROWID = (SELECT ROWID FROM webhooks WHERE chat_id = ? ORDER BY webhook_created ASC LIMIT ?, 1)"
		queryArgs = append(queryArgs, num-1)
	} else {
		queryStr = "DELETE FROM webhooks WHERE chat_id = ? AND webhook_id = ?"
		queryArgs = append(queryArgs, match[2])
	}

	// Run the query
	DB := db.GetDB()
	res, err := DB.Exec(queryStr, queryArgs...)
	if err != nil {
		nb.log.Printf("Error removing webhook for chat %d: %s\n", m.ChatId, err.Error())
		return
	}
	rows, err := res.RowsAffected()
	if err != nil {
		nb.log.Printf("Error counting affected rows after removing webhook for chat %d: %s\n", m.ChatId, err.Error())
		return
	}

	// If we haven't removed any row
	if rows < 1 {
		_, err = nb.manager.RespondToCommand(m, "I cannot find a webhook in this chat with that number or ID")
		if err != nil {
			// Log errors only
			nb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
			return
		}
		return
	}

	// All good
	_, err = nb.manager.RespondToCommand(m, "Ok, I've removed the webhook from this chat")
	if err != nil {
		// Log errors only
		nb.log.Printf("Error sending message to chat %d: %s\n", m.ChatId, err.Error())
		return
	}
}
