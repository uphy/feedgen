package cssselector

import (
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

type (
	Selection struct {
		selectFunc func() (*goquery.Selection, error)
		cache      *goquery.Selection
	}
)

func loadDocument(url string) (*goquery.Selection, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed on GET request: %w", err)
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}
	return doc.Selection, nil
}

func newSelectionFromFactory(docFunc func() (*goquery.Selection, error)) *Selection {
	return &Selection{docFunc, nil}
}

func newSelection(selection *goquery.Selection) *Selection {
	return newSelectionFromFactory(func() (*goquery.Selection, error) {
		return selection, nil
	})
}

func newSelectionFromURL(url string) (*Selection, error) {
	if s, err := loadDocument(url); err == nil {
		return newSelection(s), nil
	} else {
		return nil, fmt.Errorf("failed to load document: url=%s, err=%w", url, err)
	}
}

func (d *Selection) selection() *goquery.Selection {
	if d.cache != nil {
		return d.cache
	}

	if s, err := d.selectFunc(); err == nil {
		d.cache = s
		return s
	} else {
		panic(fmt.Errorf("failed to get selection: %w", err))
	}
}

func (d *Selection) List(selector string) ([]*Selection, error) {
	selections := make([]*Selection, 0)
	d.selection().Find(selector).Each(func(i int, s *goquery.Selection) {
		selections = append(selections, newSelection(s))
	})
	return selections, nil
}

func (d *Selection) Select(selector string) *Selection {
	return &Selection{cache: d.selection().Find(selector)}
}

func (d *Selection) Exist() bool {
	return d.selection().Length() > 0
}

func (d *Selection) Attr(attr string) string {
	if attr == "" {
		return d.String()
	}
	return d.selection().AttrOr(attr, "")
}

func (d *Selection) First() *Selection {
	return newSelection(d.selection().First())
}

func (d *Selection) HTML() (string, error) {
	return d.selection().Html()
}

func (d *Selection) String() string {
	return d.selection().Text()
}
