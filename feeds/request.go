package feeds

import (
	"errors"
	"sort"
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
		posts, err = f.RequestDockerFeed(feed)
	// Default: RSS feed
	default:
		posts, err = f.RequestRSSFeed(feed)
	}

	if err != nil {
		return nil, err
	}

	// Sort items by date, from old to new
	if posts != nil && posts.Items != nil && len(posts.Items) > 0 {
		sort.Slice(posts.Items, func(i, j int) bool {
			return posts.Items[i].PublishedParsed.Before(*posts.Items[j].PublishedParsed)
		})
	}

	return posts, nil
}
