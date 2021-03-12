# RSS bot for Telegram

This is an another Telegram bot for subscribing to RSS feeds, that you can host yourself.

In addition to RSS and Atom feeds from anywhere, you can pass an address from Docker Hub to monitor new container images.

> This project started as a hard fork of [0x111/telegram-rss-bot](https://github.com/0x111/telegram-rss-bot), created by Richard Szolár and released under a MIT license. However, the codebase has been heavily modified from the original and it includes significant improvements to reduce resource consumption (storage, bandwidth, disk I/O) and adds new features.

# Setup

## System requirements

This bot is designed to run on a Linux server with Docker. Images for both amd64 and arm64 are published on Docker Hub and on GitHub Container Registry.

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

## Run with Docker

The best way to run this bot is as a Docker container.

The container image is published to both Docker Hub and GitHub Container Registry:

- Docker Hub: `italypaleale/rss-bot`
- GitHub Container Registry: `ghcr.io/italypaleale/rss-bot`

To run with Docker:

```sh
# Replace "xxx" with your Telegram API token
docker run \
  -d \
  --restart always \
  --name rss-bot \
  -v rssdb:/data \
  -e BOT_TELEGRAMAUTHTOKEN=xxx \
  italypaleale/rss-bot:latest
```

Note the Docker volume `rssdb` mounted to `/data`, which will contain the SQLite database. Optionally, you can mount that to a local folder too.

You can pass other configuration options via environmental variables. Alternatively, you can mount a config file via a Docker volume with the flag `-v /path/to/bot-config.json:/bot-config.json`

### Telegram rate limiting

Note the [API rate limits](https://core.telegram.org/bots/faq#my-bot-is-hitting-limits-how-do-i-avoid-this) for Telegram.
