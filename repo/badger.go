package repo

import (
	"encoding/json"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/feeds"
)

type (
	BadgerRepository struct {
		db *badger.DB
	}
)

func NewBadgerRepository(file string) (*Repository, error) {
	options := badger.DefaultOptions(file)
	options.Logger = nil

	db, err := badger.Open(options)
	if err != nil {
		return nil, err
	}
	r := &BadgerRepository{db}
	return &Repository{r, r}, nil
}

func (r *BadgerRepository) PutFeed(key Key, feed *feeds.Feed) error {
	return r.put("f", key, feed)
}

func (r *BadgerRepository) GetFeed(key Key) (*feeds.Feed, error) {
	var feed feeds.Feed
	if err := r.get("f", key, &feed); err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &feed, nil
}

func (r *BadgerRepository) PutFeedItem(key Key, item *feeds.Item) error {
	return r.put("i", key, item)
}

func (r *BadgerRepository) GetFeedItem(key Key) (*feeds.Item, error) {
	var item feeds.Item
	if err := r.get("i", key, &item); err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *BadgerRepository) get(prefix string, key Key, v interface{}) error {
	var b []byte
	if err := r.db.View(func(txn *badger.Txn) error {
		key := r.key(prefix, key)
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		if err := item.Value(func(val []byte) error {
			b = val
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func (r *BadgerRepository) put(prefix string, key Key, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return r.db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry(r.key(prefix, key), b)
		entry.WithTTL(time.Hour * 24 * 30) // 30 days
		return txn.SetEntry(entry)
	})
}

func (r *BadgerRepository) key(prefix string, key Key) []byte {
	return []byte(prefix + ":" + key.Key())
}

func (r *BadgerRepository) Close() error {
	return r.db.Close()
}
