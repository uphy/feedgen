package cssselector

import (
	"fmt"
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
		URL  string `yaml:"url"`
		List string `yaml:"list"`
		Item struct {
			ID          *Field `yaml:"id"`
			Title       *Field `yaml:"title"`
			Description *Field `yaml:"description"`
			Author      *Field `yaml:"author"`
			Content     *Field `yaml:"content"`
			Link        *Field `yaml:"link"`
		} `yaml:"item"`
		Limit int `yaml:"limit"`
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

	templateContext := newContext(baseURL, doc)
	if err := templateContext.addField("id", g.config.Item.ID); err != nil {
		return fmt.Errorf("failed to initialize template field 'id': %w", err)
	}
	if err := templateContext.addField("name", g.config.Item.Title); err != nil {
		return fmt.Errorf("failed to initialize template field 'name': %w", err)
	}
	if err := templateContext.addField("description", g.config.Item.Description); err != nil {
		return fmt.Errorf("failed to initialize template field 'description': %w", err)
	}
	if err := templateContext.addField("author", g.config.Item.Author); err != nil {
		return fmt.Errorf("failed to initialize template field 'author': %w", err)
	}
	if err := templateContext.addField("content", g.config.Item.Content); err != nil {
		return fmt.Errorf("failed to initialize template field 'content': %w", err)
	}
	if err := templateContext.addField("link", g.config.Item.Link); err != nil {
		return fmt.Errorf("failed to initialize template field 'link': %w", err)
	} else {
		g.config.Item.Link.ResultMapper = func(link string) (string, error) {
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
	}

	itemContents, err := doc.List(g.config.List)
	if err != nil {
		return err
	}
	for i, itemContent := range itemContents {
		if g.config.Limit > 0 && i >= g.config.Limit {
			break
		}
		templateContext.prepare(itemContent)

		var id, title, description, author, link string
		if g.config.Item.ID != nil {
			id = g.config.Item.ID.String()
		}
		if g.config.Item.Title != nil {
			title = g.config.Item.Title.String()
		}
		if g.config.Item.Description != nil {
			description = g.config.Item.Description.String()
		}
		if g.config.Item.Author != nil {
			author = g.config.Item.Author.String()
		}
		if g.config.Item.Link != nil {
			link = g.config.Item.Link.String()
		}

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
				var content string
				if g.config.Item.Content != nil {
					content = g.config.Item.Content.String()
				}
				item.Id = id
				item.Title = title
				item.Description = description
				item.Author = &feeds.Author{
					Name: author,
				}
				item.Content = content
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
