package feeds

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/ItalyPaleAle/rss-bot/models"
)

var dockerHubMatch = regexp.MustCompile("^https:\\/\\/hub\\.docker\\.com\\/((r|repository\\/docker)\\/([a-z0-9]+)|_)\\/(.*?)$")

type dockerHubTagList struct {
	Results []struct {
		ID                  int        `json:"id"`
		Tag                 string     `json:"name"`
		LastUpdated         *time.Time `json:"last_updated"`
		LastUpdaterUsername string     `json:"last_updater_username"`
	} `json:"results"`
}

// RequestDockerFeed requests a "feed" containing the latest tags for an image on Docker Hub
func (f *Feeds) RequestDockerFeed(feed *models.Feed) (posts *gofeed.Feed, err error) {
	// Get the username and repository name
	match := dockerHubMatch.FindStringSubmatch(feed.Url)
	if len(match) < 5 {
		return nil, errors.New("invalid feed URL")
	}
	var username, repository, fullName, link string

	if match[1] == "_" {
		// From the official library
		username = "library"
	} else {
		username = match[3]
	}
	repository = match[4]

	// Create the request
	reqUrl := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/%s/tags", username, repository)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(f.ctx)
	req.Header.Set("User-Agent", "RSSBot/1.0")

	// Send the request and read the data
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, gofeed.HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	// Parse the response as JSON
	body := dockerHubTagList{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&body)
	if err != nil {
		return nil, err
	}

	// Link and full name
	if username == "library" {
		link = fmt.Sprintf("https://hub.docker.com/_/%s", repository)
		fullName = repository
	} else {
		link = fmt.Sprintf("https://hub.docker.com/r/%s/%s", username, repository)
		fullName = username + "/" + repository
	}

	// Create a Feed object with the result
	posts = &gofeed.Feed{
		Title: fmt.Sprintf("Docker Hub: %s/%s", username, repository),
		Link:  link,
	}

	if len(body.Results) == 0 {
		return posts, nil
	}

	// Iterate through the results
	posts.Items = make([]*gofeed.Item, len(body.Results))
	for i, el := range body.Results {
		posts.Items[i] = &gofeed.Item{
			Title:           el.Tag,
			PublishedParsed: el.LastUpdated,
			Author:          &gofeed.Person{Name: el.LastUpdaterUsername},
			Description:     fmt.Sprintf("Docker tag `%s/%s` updated", fullName, el.Tag),
			Link:            link,
		}
	}

	f.log.Printf("Found %d tags for image %s\n", len(posts.Items), fullName)

	return posts, nil
}
