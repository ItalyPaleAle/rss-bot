# RSS bot for Telegram

This is an another telegram bot for subscribing to RSS feeds, that you can host yourself.

> This project started as a hard fork of [0x111/telegram-rss-bot](https://github.com/0x111/telegram-rss-bot), created by Richard Szolár and released under a MIT license. However, the codebase has been heavily modified from the original and it includes significant improvements in reducing consumption of resources (storage, bandwidth, disk I/O).

# Setup

## System requirements

This bot is designed to run on a Linux server. At the moment, only the amd64 architecture is supported.

## Register your bot

To use this, you first need to register a Telegram bot by reading the [documentation](https://core.telegram.org/bots#6-botfather).

For this bot to work, you will need a token which authorizes you to use the Telegram bot APIs.

## Configuration

This bot offers some configuration options; the only one which is required is the Telegram API auth token. You can configure the bot via environmental variables or config files, or both.

### Config file

Create a config file called `bot-config.json`. You can find a template in [`bot-config.sample.json`](/bot-config.sample.json) in this repository.

You can place the config file in one of these folders (in order of decreasing priority):

- `./bot-config.json`, in the directory where the binary is located
- `$HOME/.rss-bot/bot-config.json`
- `/etc/rss-bot/bot-config.json`

Example config file, with the default values shown:

```json
{
  "TelegramAuthToken": "",
  "DBPath": "./bot.db",
  "FeedUpdateInterval": 600,
  "AllowedUsers": [],
  "TelegramAPIDebug": false
}
```

Options:

- **`TelegramAuthToken`** (string): Authentication token for the Telegram API, which you generated earlier.
- **`DBPath`** (string): Path where to store the SQLite database; by default, this is a file called `bot.db` in the directory of the binary.
- **`FeedUpdateInterval`** (integer): Number of seconds to wait before refreshing feeds; by default, that is 600, or 10 minutes.
- **`AllowedUsers`** (array of integers): If this optional value is set, only those users whose ID is in this array can interact with the bot; IDs come from Telegram. Example: `"AllowedUsers": [12345, 98765]`
- **`TelegramAPIDebug`** (boolean): If `true`, shows debug information from the Telegram APIs

### Env vars

When set, environmental variables take precedence over settings from config files.

- **`BOT_TELEGRAMAUTHTOKEN`**: Equivalent to `TelegramAuthToken` in the config file.
- **`BOT_DBPATH`**: Equivalent to `DBPath` in the config file.
- **`BOT_FEEDUPDATEINTERVAL`**: Equivalent to `FeedUpdateInterval` in the config file.
- **`BOT_ALLOWEDUSERS`**: A comma-separated list of user IDs (e.g. `BOT_ALLOWEDUSERS="12345,98765"`); this is akin to the `AllowedUsers` option in the config file.
- **`BOT_TELEGRAMAPIDEBUG`**: Equivalent to `TelegramAPIDebug` in the config file.

## Docker support

You can also run this application as a docker container.

### Docker hub

You can pull the official docker image
```bash
docker pull ruthless/telegram-rss-bot
docker run -e TelegramAuthToken="MY-TOKEN" ruthless/telegram-rss-bot
```

### Build from source

Execute the following steps:
```
git clone https://github.com/ItalyPaleAle/rss-bot
docker build -t telegram-rss-bot:latest .
docker run --name telegram-rss-bot -e TelegramAuthToken="MY-TOKEN" -d telegram-rss-bot:latest
```

### Telegram rate limiting

Note the [API rate limis](https://core.telegram.org/bots/faq#my-bot-is-hitting-limits-how-do-i-avoid-this) for Telegram.
