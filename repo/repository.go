package repo

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/gorilla/feeds"
)

type (
	FeedRepository interface {
		io.Closer
		PutFeed(Key, *feeds.Feed) error
		GetFeed(Key) (*feeds.Feed, error)
	}
	FeedItemRepository interface {
		io.Closer
		PutFeedItem(Key, *feeds.Item) error
		GetFeedItem(Key) (*feeds.Item, error)
	}
	Repository struct {
		Feed FeedRepository
		Item FeedItemRepository
	}
	Key interface {
		Key() string
	}
	idKey        string
	generatedKey []string
)

func IDKey(id string) Key {
	return idKey(id)
}

func GeneratedKey(src ...string) Key {
	return generatedKey(src)
}

func (k idKey) Key() string {
	return string(k)
}

func (k generatedKey) Key() string {
	s := strings.Join([]string(k), "-")
	b := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", b)
}

func (r *Repository) Close() error {
	feedErr := r.Feed.Close()
	itemErr := r.Item.Close()
	if feedErr != nil || itemErr != nil {
		return fmt.Errorf("failed to close: feedErr=%w, itemErr%w", feedErr, itemErr)
	}
	return nil
}
