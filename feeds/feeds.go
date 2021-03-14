package feeds

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/ItalyPaleAle/rss-bot/db"
)

// Post represents a post in the feed
type Post struct {
	Title string
	Link  string
	Date  time.Time
	Photo string
}

// UpdateMessage is the message that needs to be sent to subscribers for new posts
type UpdateMessage struct {
	Feed   *db.Feed
	Post   Post
	ChatId int64
}

// Timeout for HTTP requests
const requestTimeout = 20 * time.Second

// Error returned when we're already subscribed
var ErrAlreadySubscribed = errors.New("already_subscribed")

// Feeds is an object that manages feeds and subscriptions
type Feeds struct {
	ctx       context.Context
	log       *log.Logger
	semaphore chan int
	waiting   chan int
	updateCh  chan<- UpdateMessage
	client    *http.Client
}

// Init the object
func (f *Feeds) Init(ctx context.Context) (err error) {
	f.ctx = ctx

	// Init the logger
	f.log = log.New(os.Stdout, "feeds: ", log.Ldate|log.Ltime|log.LUTC)

	// Init the update semaphore and waiting channels
	f.semaphore = make(chan int, 1)
	f.waiting = make(chan int, 1)

	// Init the HTTP client
	f.client = &http.Client{
		Timeout: requestTimeout,
	}

	return nil
}

// AddSubscription subscribes a chat to a feed, adding the feed if required
func (f *Feeds) AddSubscription(url string, chatId int64) (*Post, error) {
	if chatId == 0 {
		return nil, errors.New("Empty chat ID")
	}

	DB := db.GetDB()

	// Begin a transaction
	tx, err := DB.Beginx()
	if err != nil {
		f.log.Println("Error starting a transaction:", err)
		return nil, err
	}
	defer tx.Rollback()

	// Check if the feed exists already
	feed, err := f.GetFeedByURL(url, tx)
	if err != nil {
		// Error was already logged
		return nil, err
	}

	// If the feed doesn't exist, add it
	if feed == nil || feed.ID < 1 {
		feed, err = f.AddFeed(url, tx)
		if err != nil {
			// Error was already logged
			return nil, err
		}
	}

	// Check if the subscription already exists
	subscription := &db.Subscription{}
	err = tx.Get(subscription, "SELECT subscription_id FROM subscriptions WHERE feed_id = ? AND chat_id = ? LIMIT 1", feed.ID, chatId)
	// There should be an error, and it should be ErrNoRows
	if err == nil {
		return nil, ErrAlreadySubscribed
	} else if err != sql.ErrNoRows {
		// Another error, needs to be handled
		f.log.Println("Error querying the database:", err)
		return nil, err
	}

	// Add the subscription
	_, err = tx.Exec("INSERT INTO subscriptions (feed_id, chat_id) VALUES (?, ?)", feed.ID, chatId)
	if err != nil {
		f.log.Println("Error querying the database:", err)
		return nil, err
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		f.log.Println("Error while committing the transaction:", err)
		return nil, err
	}

	f.log.Printf("Added feed %s (ID %d) to chat %d", url, feed.ID, chatId)

	// Get the Post object from the feed
	post := &Post{
		Title: feed.LastPostTitle,
		Link:  feed.LastPostLink,
		Date:  feed.LastPostDate,
		Photo: feed.LastPostPhoto,
	}
	return post, nil
}

// DeleteSubscription removes a subscription to a feed
// If this is the last subscription to a feed, remove the feed too
func (f *Feeds) DeleteSubscription(feedId int64, chatId int64) error {
	DB := db.GetDB()

	// Begin a transaction
	tx, err := DB.Beginx()
	if err != nil {
		f.log.Println("Error starting a transaction:", err)
		return err
	}
	defer tx.Rollback()

	// Delete the subscription
	_, err = tx.Exec("DELETE FROM subscriptions WHERE feed_id = ? AND chat_id = ?", feedId, chatId)
	if err != nil {
		f.log.Println("Error querying the database:", err)
		return err
	}

	// Check if there are other subscriptions for this feed
	subscription := &db.Subscription{}
	err = tx.Get(subscription, "SELECT subscription_id FROM subscriptions WHERE feed_id = ?", feedId)
	if err != nil {
		// If there are no more rows, delete the feed
		if err == sql.ErrNoRows {
			_, err = tx.Exec("DELETE FROM feeds WHERE feed_id = ?", feedId)
			if err != nil {
				f.log.Println("Error querying the database:", err)
				return err
			}
		} else {
			// Another error, needs to be handled
			f.log.Println("Error querying the database:", err)
			return err
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		f.log.Println("Error while committing the transaction:", err)
		return err
	}

	return nil
}

// ListSubscriptions lists all subscriptions for a chat
func (f *Feeds) ListSubscriptions(chatId int64) ([]db.Feed, error) {
	DB := db.GetDB()

	// Query the DB
	rows := []db.Feed{}
	err := DB.Select(&rows, "SELECT feeds.* FROM feeds, subscriptions WHERE chat_id = ? AND feeds.feed_id = subscriptions.feed_id ORDER BY feed_url ASC", chatId)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows
			return nil, nil
		}
		f.log.Println("Error querying the database:", err)
		return nil, err
	}

	return rows, nil
}

// GetFeedByURL returns a feed from its URL, or 0 if it's not present
// The transaction is optional
func (f *Feeds) GetFeedByURL(url string, tx *sqlx.Tx) (*db.Feed, error) {
	// Use a transaction if we have one
	var querier sqlx.Ext = db.GetDB()
	if tx != nil {
		querier = tx
	}

	// Run the query
	feed := &db.Feed{}
	err := sqlx.Get(querier, feed, "SELECT * FROM feeds WHERE feed_url = ?", url)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows found, so record doesn't exist
			return nil, nil
		}
		f.log.Println("Error querying the database:", err)
		return nil, err
	}

	return feed, nil
}

// AddFeed adds a new feed
// The transaction is optional
func (f *Feeds) AddFeed(url string, tx *sqlx.Tx) (*db.Feed, error) {
	// Use a transaction if we have one
	var querier sqlx.Ext = db.GetDB()
	if tx != nil {
		querier = tx
	}

	// Get the feed to both validate it and to get the latest entry
	f.log.Println("Fetching feed", url)
	feed := &db.Feed{
		Url:   url,
		Title: url,
	}
	posts, err := f.RequestFeed(feed)
	if err != nil {
		f.log.Printf("Error while fetching feed %d: %s\n", feed.ID, err)
		return nil, err
	}

	// Get the most recent, valid entry
	if posts != nil && posts.Items != nil {
		for _, el := range posts.Items {
			// Check if this is newer than the one stored
			if el != nil && el.PublishedParsed != nil && el.PublishedParsed.After(feed.LastPostDate) {
				p := Post{
					Title: el.Title,
					Link:  el.Link,
					Date:  *el.PublishedParsed,
				}

				// Request the metadata for the post
				f.RequestMetadata(&p)

				feed.LastPostTitle = p.Title
				feed.LastPostLink = p.Link
				feed.LastPostDate = p.Date
				feed.LastPostPhoto = p.Photo
			}
		}
	}

	// Get the feed's title
	if posts != nil && posts.Title != "" {
		feed.Title = posts.Title
	}

	// Add the feed to the database
	res, err := querier.Exec("INSERT INTO feeds (feed_url, feed_title, feed_last_modified, feed_etag, feed_last_post_title, feed_last_post_link, feed_last_post_date, feed_last_post_photo) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", feed.Url, feed.Title, feed.LastModified, feed.ETag, feed.LastPostTitle, feed.LastPostLink, feed.LastPostDate, feed.LastPostPhoto)
	if err != nil {
		f.log.Println("Error inserting in the database:", err)
		return nil, err
	}
	feed.ID, err = res.LastInsertId()
	if err != nil {
		f.log.Println("Error getting the last rowid:", err)
		return nil, err
	}
	if feed.ID < 1 {
		return nil, errors.New("Empty feed ID")
	}
	f.log.Printf("Added feed %s with ID %d", url, feed.ID)

	return feed, nil
}
