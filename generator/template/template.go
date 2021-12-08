package template

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/uphy/feedgen/config"
	"github.com/uphy/feedgen/generator"
	"github.com/uphy/feedgen/repo"
	tmpl "github.com/uphy/feedgen/template"

	"github.com/gorilla/feeds"
)

type (
	TemplateFeedGeneratorConfig struct {
		URL   tmpl.TemplateField `yaml:"url"`
		Feed  FeedConfig         `yaml:"feed"`
		List  tmpl.TemplateField `yaml:"list"`
		Item  ItemConfig         `yaml:"item"`
		Limit int                `yaml:"limit"`
	}
	FeedConfig struct {
		ID    tmpl.TemplateField `yaml:"id"`
		Title tmpl.TemplateField `yaml:"title"`
		Link  struct {
			Href tmpl.TemplateField `yaml:"href"`
		} `yaml:"link"`
		Description tmpl.TemplateField `yaml:"description"`
		Author      struct {
			Name  tmpl.TemplateField `yaml:"name"`
			Email tmpl.TemplateField `yaml:"email"`
		} `yaml:"author"`
	}
	ItemConfig struct {
		ID          tmpl.TemplateField `yaml:"id"`
		Title       tmpl.TemplateField `yaml:"title"`
		Description tmpl.TemplateField `yaml:"description"`
		Author      tmpl.TemplateField `yaml:"author"`
		Content     tmpl.TemplateField `yaml:"content"`
		Link        struct {
			HREF tmpl.TemplateField `yaml:"href"`
		} `yaml:"link"`
		Keys []string `yaml:"keys"`
	}
	TemplateFeedGenerator struct {
		config *TemplateFeedGeneratorConfig
	}
)

func (g *TemplateFeedGenerator) LoadOptions(options config.GeneratorOptions) error {
	var c TemplateFeedGeneratorConfig
	if err := options.Unmarshal(&c); err != nil {
		return err
	}
	g.config = &c
	return nil
}

func (g *TemplateFeedGenerator) Generate(feed *feeds.Feed, context *generator.Context) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("failed to generate: %v", rec)
		}
	}()
	err = g.generate(feed, context)
	return
}

func (g *TemplateFeedGenerator) generate(feed *feeds.Feed, context *generator.Context) error {
	context.TemplateContext.AddFuncs(map[string]interface{}{
		"ReplaceAll": func(old, new, s string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"Attr": func(attr string, input *Selection) string {
			return input.Attr(attr)
		},
		"Text": func(input *Selection) string {
			return input.Text()
		},
	})

	baseURL, err := url.Parse(g.config.URL.MustEvaluate(context.TemplateContext))
	if err != nil {
		return err
	}
	context.TemplateContext.Set("URL", baseURL.String())

	doc, err := newSelectionFromURL(baseURL.String())
	if err != nil {
		return err
	}
	context.TemplateContext.Set("Content", doc)

	/*
	 * Feed
	 */
	feed.Id = g.config.Feed.ID.MustEvaluate(context.TemplateContext)
	feed.Title = g.config.Feed.Title.MustEvaluate(context.TemplateContext)
	feed.Link = &feeds.Link{
		Href: g.config.Feed.Link.Href.MustEvaluate(context.TemplateContext),
	}
	feed.Author = &feeds.Author{
		Name:  g.config.Feed.Author.Name.MustEvaluate(context.TemplateContext),
		Email: g.config.Feed.Author.Email.MustEvaluate(context.TemplateContext),
	}
	feed.Description = g.config.Feed.Description.MustEvaluate(context.TemplateContext)
	feed.Created = time.Now()
	feed.Updated = feed.Created

	/*
	 * Items
	 */
	itemContents, err := doc.List(g.config.List.MustEvaluate(context.TemplateContext))
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
	for i, itemContent := range itemContents {
		if g.config.Limit > 0 && i >= g.config.Limit {
			break
		}
		itemScopeContext := context.TemplateContext.Child()
		itemScopeContext.Set("ItemContent", itemContent)
		itemScopeContext.Set("LinkContent", newSelectionFromFactory(func() (*goquery.Selection, error) {
			if g.config.Item.Link.HREF.IsDefined() {
				return loadDocument(g.config.Item.Link.HREF.MustEvaluate(itemScopeContext))
			}
			return nil, fmt.Errorf("'link' not defined in config file")
		}))

		id := g.config.Item.ID.MustEvaluate(itemScopeContext)
		title := g.config.Item.Title.MustEvaluate(itemScopeContext)
		description := g.config.Item.Description.MustEvaluate(itemScopeContext)
		author := g.config.Item.Author.MustEvaluate(itemScopeContext)
		link := g.config.Item.Link.HREF.MustEvaluate(itemScopeContext)

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
				item.Content = g.config.Item.Content.MustEvaluate(itemScopeContext)
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
