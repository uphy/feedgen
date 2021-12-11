package converter

import "github.com/gorilla/feeds"

type atomConverter struct {
}

func (c *atomConverter) Convert(feed *feeds.Feed) (*Result, error) {
	atom, err := feed.ToAtom()
	if err != nil {
		return nil, err
	}
	return newResult("application/xml", atom), nil
}
