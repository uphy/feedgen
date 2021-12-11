package converter

import "github.com/gorilla/feeds"

type (
	Converter interface {
		Convert(*feeds.Feed) (*Result, error)
	}
	Result struct {
		ContentType string
		Result      string
	}
)

func GetConverter(name string) Converter {
	switch name {
	case "rss":
		return &rssConverter{}
	case "atom":
		return &atomConverter{}
	case "html":
		return &htmlConverter{}
	}
	return nil
}

func newResult(contentType, result string) *Result {
	return &Result{contentType, result}
}
