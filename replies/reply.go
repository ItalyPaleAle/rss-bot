package replies

import (
	"strconv"
	"strings"

	"github.com/0x111/telegram-rss-bot/models"

	tgbotapi "github.com/dilfish/telegram-bot-api-up"
)

// ListOfFeeds sends the list of feeds to for the command /list
func ListOfFeeds(botAPI *tgbotapi.BotAPI, feeds *[]models.Feed, chatid int64, replyMessage int) {
	var txt string
	if len(*feeds) == 0 {
		txt = "There is currently no feed added to the list for this Room\\!\n"
	} else {
		txt = "Here is the list of your added Feeds for this Room: \n"
		for _, feed := range *feeds {
			txt += "[\\#" + strconv.Itoa(feed.ID) + "] *" + FilterMessageChars(feed.Name) + "*: " + FilterMessageChars(feed.Url) + "\n"
		}
	}

	msg := tgbotapi.NewMessage(chatid, txt)
	if replyMessage != 0 {
		msg.ReplyToMessageID = replyMessage
	}

	msg.ParseMode = "markdownv2"
	msg.DisableWebPagePreview = true

	botAPI.Send(msg)
}

// SimpleMessage sends a simple message
func SimpleMessage(botAPI *tgbotapi.BotAPI, chatid int64, replyMessage int, text string) error {
	msg := tgbotapi.NewMessage(chatid, text)

	if replyMessage != 0 {
		msg.ReplyToMessageID = replyMessage
	}

	msg.ParseMode = "markdownv2"
	msg.DisableWebPagePreview = false

	_, err := botAPI.Send(msg)

	if err != nil {
		return err
	}

	return nil
}

// FilterMessageChars escapes some characters in the message
func FilterMessageChars(msg string) string {
	var markdownEscaper = strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)

	return markdownEscaper.Replace(msg)
}
