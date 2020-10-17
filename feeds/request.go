package feeds

import (
	"errors"
	"net/http"
	"time"

	"github.com/Songmu/go-httpdate"
	"github.com/mmcdole/gofeed"

	"github.com/0x111/telegram-rss-bot/models"
)

// RequestFeed requests a feed and parses it with gofeed
// We're using this rather than gofeed.ParseURL to have more control on the request
func (f *Feeds) RequestFeed(feed *models.Feed) (posts *gofeed.Feed, err error) {
	if feed.Url == "" {
		return nil, errors.New("empty feed URL")
	}

	// Create the request
	req, err := http.NewRequest("GET", feed.Url, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(f.ctx)
	req.Header.Set("User-Agent", "RSSBot/1.0")
	if !feed.LastModified.IsZero() {
		req.Header.Set("If-Modified-Since", feed.LastModified.Format(time.RFC1123Z))
	}
	if feed.ETag != "" {
		req.Header.Set("If-None-Match", feed.ETag)
	}

	// Send the request and read the data
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 304: not modified, so return an empty list
		if resp.StatusCode == http.StatusNotModified {
			f.log.Printf("Feed %d not modified\n", feed.ID)
			return nil, nil
		}
		return nil, gofeed.HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	// Get the ETag and Last-Modified headers
	etag := resp.Header.Get("ETag")
	if etag != "" {
		feed.ETag = etag
	}
	lastModified := resp.Header.Get("Last-Modified")
	if lastModified != "" {
		d, err := httpdate.Str2Time(lastModified, nil)
		if err == nil && !d.IsZero() {
			feed.LastModified = d
		}
	}

	// Parse the feed
	fp := gofeed.NewParser()
	return fp.Parse(resp.Body)
}
