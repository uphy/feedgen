package converter

import "github.com/gorilla/feeds"

type rssConverter struct {
}

func (c *rssConverter) Convert(feed *feeds.Feed) (*Result, error) {
	rss, err := feed.ToRss()
	if err != nil {
		return nil, err
	}
	return newResult("application/xml", rss), nil
}
