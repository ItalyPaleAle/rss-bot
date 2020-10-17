package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/0x111/telegram-rss-bot/feeds"
	"github.com/0x111/telegram-rss-bot/replies"

	tgbotapi "github.com/dilfish/telegram-bot-api-up"
)

// Help text
const helpTxt = `
Avaliable commands:
/add %FeedName %URL - With this you can add a new feed for the current channel, both the name and the url parameters are required
/list - With this command you are able to list all the existing feeds with their ID numbers
/delete %ID - With this command you are able to delete an added feed if you do not need it anymore. The ID parameter is required and you can get it from the /list command 
`

// Welcome text
const welcomeTxt = `
Welcome to this bot!

` + helpTxt

// Commands functions which are executed upon receiving a command

// AddCommand handles the /add command
func AddCommand(Bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	commandArguments := strings.Split(update.Message.CommandArguments(), " ")
	userid := update.Message.From.ID
	chatid := update.Message.Chat.ID

	if len(commandArguments) < 2 {
		replies.SimpleMessage(Bot, chatid, replyMessageId(update), "Not enough arguments\\. We need \"/add name url\"")
		return
	}

	feedName := commandArguments[0]
	feedUrl := commandArguments[1]

	err := feeds.AddFeed(Bot, feedName, feedUrl, chatid, userid)
	txt := ""

	if err == nil {
		txt = fmt.Sprintf("The feed with the url [%s] was successfully added to this channel\\!", replies.FilterMessageChars(feedUrl))
		replies.SimpleMessage(Bot, chatid, replyMessageId(update), txt)
	}
}

// ListCommand handles the /list command
func ListCommand(Bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	chatid := update.Message.Chat.ID
	userid := update.Message.From.ID
	feedres, err := feeds.ListFeeds(userid, chatid)

	if err != nil {
		panic(err)
	}

	replies.ListOfFeeds(Bot, feedres, chatid, replyMessageId(update))
}

// DeleteCommand handles the /delete command
func DeleteCommand(Bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	commandArguments := strings.Split(update.Message.CommandArguments(), " ")

	if len(commandArguments) < 1 {
		panic("Not enough arguments\\. We need \"/delete id\"")
	}

	feedid, _ := strconv.Atoi(commandArguments[0])
	chatid := update.Message.Chat.ID
	userid := update.Message.From.ID
	err := feeds.DeleteFeedByID(feedid, chatid, userid)

	if err != nil {
		txt := fmt.Sprintf("There is no feed with the id [%d]\\!", feedid)
		replies.SimpleMessage(Bot, chatid, replyMessageId(update), txt)
		return
	}

	txt := fmt.Sprintf("The feed with the id [%d] was successfully deleted\\!", feedid)
	replies.SimpleMessage(Bot, chatid, replyMessageId(update), txt)
}

// StartCommand handles the /start command
func StartCommand(Bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	replies.SimpleMessage(Bot, update.Message.Chat.ID, replyMessageId(update), replies.FilterMessageChars(welcomeTxt))
}

// HelpCommand handles the /help command
func HelpCommand(Bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	replies.SimpleMessage(Bot, update.Message.Chat.ID, replyMessageId(update), replies.FilterMessageChars(helpTxt))
}

func replyMessageId(update *tgbotapi.Update) (replyMessage int) {
	if !update.Message.Chat.IsPrivate() {
		replyMessage = update.Message.MessageID
	}
	return
}
