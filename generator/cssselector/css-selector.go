package cssselector

import (
	"net/url"
	"strings"
	"time"

	"github.com/uphy/feedgen/config"
	"github.com/uphy/feedgen/generator"
	"github.com/uphy/feedgen/repo"

	"github.com/gorilla/feeds"
)

type (
	CSSSelectorConfig struct {
		URL   string     `yaml:"url"`
		Feed  FeedConfig `yaml:"feed"`
		List  string     `yaml:"list"`
		Item  ItemConfig `yaml:"item"`
		Limit int        `yaml:"limit"`
	}
	FeedConfig struct {
		ID    Field `yaml:"id"`
		Title Field `yaml:"title"`
		Link  struct {
			Href Field `yaml:"href"`
		} `yaml:"link"`
		Description Field `yaml:"description"`
		Author      struct {
			Name  Field `yaml:"name"`
			Email Field `yaml:"email"`
		} `yaml:"author"`
	}
	ItemConfig struct {
		ID          Field `yaml:"id"`
		Title       Field `yaml:"title"`
		Description Field `yaml:"description"`
		Author      Field `yaml:"author"`
		Content     Field `yaml:"content"`
		Link        struct {
			HREF Field `yaml:"href"`
		} `yaml:"link"`
		Keys []string `yaml:"keys"`
	}
	CSSSelectorFeedGenerator struct {
		config *CSSSelectorConfig
	}
)

func (g *CSSSelectorFeedGenerator) LoadOptions(options config.GeneratorOptions) error {
	var c CSSSelectorConfig
	if err := options.Unmarshal(&c); err != nil {
		return err
	}
	g.config = &c
	return nil
}

func (g *CSSSelectorFeedGenerator) Generate(feed *feeds.Feed, context *generator.Context) error {
	baseURL, err := url.Parse(g.config.URL)
	if err != nil {
		return err
	}

	doc, err := newSelectionFromURL(g.config.URL)
	if err != nil {
		return err
	}

	templateContext, err := newContext(baseURL, doc, &g.config.Feed, &g.config.Item)
	if err != nil {
		return err
	}

	g.config.Item.Link.HREF.ResultMapper = func(link string) (string, error) {
		link = strings.TrimSpace(link)
		linkURL, err := url.Parse(link)
		if err != nil {
			return "", err
		}
		if !linkURL.IsAbs() {
			link = baseURL.ResolveReference(linkURL).String()
		}
		return link, nil
	}

	/*
	 * Feed
	 */
	feed.Id = g.config.Feed.ID.String()
	feed.Title = g.config.Feed.Title.String()
	feed.Link = &feeds.Link{
		Href: g.config.Feed.Link.Href.String(),
	}
	feed.Author = &feeds.Author{
		Name:  g.config.Feed.Author.Name.String(),
		Email: g.config.Feed.Author.Email.String(),
	}
	feed.Description = g.config.Feed.Description.String()
	feed.Created = time.Now()
	feed.Updated = feed.Created

	/*
	 * Items
	 */
	itemContents, err := doc.List(g.config.List)
	if err != nil {
		return err
	}
	for i, itemContent := range itemContents {
		if g.config.Limit > 0 && i >= g.config.Limit {
			break
		}
		templateContext.prepare(itemContent)

		id := g.config.Item.ID.String()
		title := g.config.Item.Title.String()
		description := g.config.Item.Description.String()
		author := g.config.Item.Author.String()
		link := g.config.Item.Link.HREF.String()

		// Get cache or create new feed item
		var key repo.Key
		if id != "" {
			key = repo.IDKey(id)
		} else {
			key = repo.GeneratedKey(title, description, author)
		}
		if item, err := context.Repository.Item.GetFeedItem(key); err == nil {
			if item == nil {
				item = new(feeds.Item)
				item.Id = id
				item.Title = title
				item.Description = description
				item.Author = &feeds.Author{
					Name: author,
				}
				item.Content = g.config.Item.Content.String()
				item.Link = &feeds.Link{
					Href: link,
				}
				item.Created = time.Now()
				item.Updated = item.Created
				if err := context.Repository.Item.PutFeedItem(key, item); err != nil {
					return err
				}
			} else {
				var updated = false
				if title != "" && item.Title != title {
					item.Title = title
					updated = true
				}
				if description != "" && item.Description != description {
					item.Description = description
					updated = true
				}
				if updated {
					item.Updated = time.Now()
					if err := context.Repository.Item.PutFeedItem(key, item); err != nil {
						return err
					}
				}
			}
			feed.Items = append(feed.Items, item)
		} else {
			return err
		}
	}
	return nil
}
