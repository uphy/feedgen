package template

import (
	"errors"
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
		Author      struct {
			Name tmpl.TemplateField `yaml:"name"`
		} `yaml:"author"`
		Content tmpl.TemplateField `yaml:"content"`
		Link    struct {
			HREF tmpl.TemplateField `yaml:"href"`
		} `yaml:"link"`
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

func (g *TemplateFeedGenerator) Generate(context *generator.Context) (feed *feeds.Feed, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("failed to generate: %v", rec)
			feed = nil
		}
	}()
	feed, err = g.generate(context)
	return
}

func (g *TemplateFeedGenerator) generate(context *generator.Context) (*feeds.Feed, error) {
	currentTemplateContext := context.TemplateContext
	context.TemplateContext.AddFuncs(map[string]interface{}{
		"ReplaceAll": func(old, new, s string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"Attr": func(attr string, input *Selection) string {
			return input.Attr(attr)
		},
		"Text": func(input interface{}) string {
			return toString(currentTemplateContext, input)
		},
	})

	baseURL, err := url.Parse(g.config.URL.MustEvaluate(currentTemplateContext))
	if err != nil {
		return nil, err
	}
	currentTemplateContext.Set("URL", baseURL.String())

	doc, err := newSelectionFromURL(baseURL.String())
	if err != nil {
		return nil, err
	}
	currentTemplateContext.Set("Content", doc)

	/*
	 * Feed
	 */
	feed := new(feeds.Feed)
	currentTemplateContext.Set("Feed", feed)
	feed.Id = g.config.Feed.ID.MustEvaluate(currentTemplateContext)
	if feedCache, err := context.Repository.Feed.GetFeed(repo.IDKey(feed.Id)); err == nil {
		if feedCache == nil {
			feed.Title = g.config.Feed.Title.MustEvaluate(currentTemplateContext)
			feed.Link = &feeds.Link{
				Href: g.config.Feed.Link.Href.MustEvaluate(currentTemplateContext),
			}
			feed.Author = &feeds.Author{
				Name:  g.config.Feed.Author.Name.MustEvaluate(currentTemplateContext),
				Email: g.config.Feed.Author.Email.MustEvaluate(currentTemplateContext),
			}
			feed.Description = g.config.Feed.Description.MustEvaluate(currentTemplateContext)
			feed.Created = time.Now()
			feed.Updated = feed.Created
		} else {
			feed = feedCache
			currentTemplateContext.Set("Feed", feed)
		}
	} else {
		return nil, err
	}

	/*
	 * Items
	 */
	currentTemplateContext.Set("Item", g.config.Item)
	itemContents, err := doc.List(g.config.List.MustEvaluate(currentTemplateContext))
	if err != nil {
		return nil, err
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

		// prepare template context for the item
		currentTemplateContext = context.TemplateContext.Child()
		currentTemplateContext.Set("ItemContent", itemContent)
		currentTemplateContext.Set("LinkContent", newSelectionFromFactory(func() (*goquery.Selection, error) {
			if g.config.Item.Link.HREF.IsDefined() {
				return loadDocument(g.config.Item.Link.HREF.MustEvaluate(currentTemplateContext))
			}
			return nil, fmt.Errorf("'link' not defined in config file")
		}))

		// Evaluate 'id' first for getting cache.
		id := g.config.Item.ID.MustEvaluate(currentTemplateContext)
		if len(id) == 0 {
			id = g.config.Item.Link.HREF.MustEvaluate(currentTemplateContext)
			if len(id) == 0 {
				return nil, errors.New("'id' or 'link.href' is required")
			}
		}

		// Get cache or create new feed item
		key := repo.IDKey(id)
		if item, err := context.Repository.Item.GetFeedItem(key); err == nil {
			if item == nil {
				item = new(feeds.Item)
				item.Id = id
				item.Title = g.config.Item.Title.MustEvaluate(currentTemplateContext)
				item.Description = g.config.Item.Description.MustEvaluate(currentTemplateContext)
				item.Author = &feeds.Author{
					Name: g.config.Item.Author.Name.MustEvaluate(currentTemplateContext),
				}
				link := g.config.Item.Link.HREF.MustEvaluate(currentTemplateContext)
				item.Content = g.config.Item.Content.MustEvaluate(currentTemplateContext)
				item.Link = &feeds.Link{
					Href: link,
				}
				item.Created = time.Now()
				item.Updated = item.Created
				if err := context.Repository.Item.PutFeedItem(key, item); err != nil {
					return nil, err
				}
			}
			feed.Items = append(feed.Items, item)
		} else {
			return nil, err
		}
	}
	return feed, nil
}

func toString(templateContext *tmpl.TemplateContext, i interface{}) string {
	switch v := i.(type) {
	case *Selection:
		return v.Text()
	case tmpl.TemplateField:
		return v.MustEvaluate(templateContext)
	}
	return fmt.Sprint(i)
}
