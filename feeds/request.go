package feeds

import (
	"errors"
	"strings"

	"github.com/mmcdole/gofeed"

	"github.com/ItalyPaleAle/rss-bot/models"
)

// RequestFeed requests a feed of any kind
func (f *Feeds) RequestFeed(feed *models.Feed) (posts *gofeed.Feed, err error) {
	if feed.Url == "" {
		return nil, errors.New("empty feed URL")
	}

	// Check the type of feed
	switch {
	// Docker Hub
	case strings.HasPrefix(feed.Url, "https://hub.docker.com/"):
		return f.RequestDockerFeed(feed)
	// Default: RSS feed
	default:
		return f.RequestRSSFeed(feed)
	}
}
