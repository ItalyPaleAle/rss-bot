package feeds

import (
	"errors"
	"net/http"
	"time"

	"github.com/Songmu/go-httpdate"
	"github.com/mmcdole/gofeed"

	"github.com/ItalyPaleAle/rss-bot/models"
)

// RequestRSSFeed requests a RSS feed and parses it with gofeed
// We're using this rather than gofeed.ParseURL to have more control on the request
func (f *Feeds) RequestRSSFeed(feed *models.Feed) (posts *gofeed.Feed, err error) {
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
	posts, err = fp.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	// Iterate through the results
	n := 0
	for _, el := range posts.Items {
		// Skip items with an invalid date
		if el.PublishedParsed == nil || el.PublishedParsed.IsZero() {
			f.log.Printf("Error in feed %s: skipping entry with invalid date '%s' (error: %s)\n", feed.Url, el.Published, err)
			continue
		}

		// Skip items with an empty title
		if el.Title == "" {
			f.log.Printf("Error in feed %s: skipping entry with empty title\n", feed.Url)
			continue
		}

		// Keep the item
		posts.Items[n] = el
		n++
	}
	posts.Items = posts.Items[:n]

	if feed.ID > 0 {
		f.log.Printf("Found %d posts in feed %d\n", len(posts.Items), feed.ID)
	} else {
		f.log.Printf("Found %d posts in new feed\n", len(posts.Items))
	}

	return posts, nil
}
