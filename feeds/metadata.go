package feeds

import (
	"net/http"

	"github.com/mmcdole/gofeed"
	opengraph "github.com/otiai10/opengraph/v2"
)

// RequestMetadata requests the web page to get the title, description, and image from the page's metadata itself
// This method updates the value of the post argument as a side effect
// Errors are logged only and then ignored
func (f *Feeds) RequestMetadata(post *Post) {
	if post.Link == "" {
		return
	}

	// Wrapping this in a method that returns an error
	err := f.doRequestMetadata(post)
	if err != nil {
		f.log.Printf("Error while requesting the page %s: %s\n", post.Link, err)
		return
	}
}

// Performs
func (f *Feeds) doRequestMetadata(post *Post) (err error) {
	// Request the web page
	req, err := http.NewRequest("GET", post.Link, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(f.ctx)
	req.Header.Set("User-Agent", "RSSBot/1.0")
	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return gofeed.HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	// Read the response and extract the OpenGraph tags
	ogp := &opengraph.OpenGraph{}
	err = ogp.Parse(resp.Body)
	if err != nil {
		return err
	}
	err = ogp.ToAbs()
	if err != nil {
		return err
	}

	// Update the feed's data with information from OpenGraph
	if ogp.Title != "" {
		post.Title = ogp.Title
	}
	if len(ogp.Image) > 0 {
		post.Photo = ogp.Image[0].URL
	}

	return nil
}
