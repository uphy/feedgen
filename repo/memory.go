package repo

import (
	"encoding/json"

	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/feeds"
)

type (
	MemoryRepository struct {
		keyValue map[string][]byte
	}
)

func NewMemoryRepository() *Repository {
	r := &MemoryRepository{make(map[string][]byte)}
	return &Repository{r, r}
}

func (r *MemoryRepository) PutFeed(key Key, feed *feeds.Feed) error {
	return r.put("f", key, feed)
}

func (r *MemoryRepository) GetFeed(key Key) (*feeds.Feed, error) {
	var feed feeds.Feed
	if err := r.get("f", key, &feed); err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &feed, nil
}

func (r *MemoryRepository) PutFeedItem(key Key, item *feeds.Item) error {
	return r.put("i", key, item)
}

func (r *MemoryRepository) GetFeedItem(key Key) (*feeds.Item, error) {
	var item feeds.Item
	if err := r.get("i", key, &item); err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *MemoryRepository) get(prefix string, key Key, v interface{}) error {
	if value, exist := r.keyValue[r.key(prefix, key)]; exist {
		return json.Unmarshal(value, v)
	} else {
		return badger.ErrKeyNotFound
	}
}

func (r *MemoryRepository) put(prefix string, key Key, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	r.keyValue[r.key(prefix, key)] = b
	return nil
}

func (r *MemoryRepository) key(prefix string, key Key) string {
	return prefix + ":" + key.Key()
}

func (r *MemoryRepository) Close() error {
	r.keyValue = make(map[string][]byte)
	return nil
}
