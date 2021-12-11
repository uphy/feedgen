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
	"github.com/uphy/feedgen/generator/source"
	"github.com/uphy/feedgen/repo"
	tmpl "github.com/uphy/feedgen/template"

	"github.com/gorilla/feeds"
)

type (
	TemplateFeedGeneratorConfig struct {
		Source *source.Source     `yaml:"source"`
		Feed   FeedConfig         `yaml:"feed"`
		List   tmpl.TemplateField `yaml:"list"`
		Item   ItemConfig         `yaml:"item"`
		Limit  int                `yaml:"limit"`
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
	templateContext := context.TemplateContext
	context.TemplateContext.AddFuncs(map[string]interface{}{
		"ReplaceAll": func(old, new, s string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"Attr": func(attr string, input *Selection) string {
			return input.Attr(attr)
		},
		"Text": func(input interface{}) string {
			return toString(templateContext, input)
		},
	})

	/*
	 * Source
	 */
	var baseURL *url.URL
	var doc *Selection
	if u, d, err := g.loadSource(templateContext); err == nil {
		baseURL = u
		doc = d
	} else {
		return nil, err
	}

	/*
	 * Feed
	 */
	var feed *feeds.Feed
	if f, err := g.loadFeed(templateContext, context.Repository); err == nil {
		feed = f
	} else {
		return nil, err
	}

	/*
	 * Items
	 */
	templateContext.Set("Item", g.config.Item)
	itemContents, err := doc.List(g.config.List.MustEvaluate(templateContext))
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
	itemTemplateContext := templateContext
	for i, itemContent := range itemContents {
		if g.config.Limit > 0 && i >= g.config.Limit {
			break
		}
		templateContext = itemTemplateContext.Child()
		if item, err := g.loadItem(templateContext, context.Repository, itemContent); err == nil {
			feed.Items = append(feed.Items, item)
		} else {
			return nil, err
		}
	}
	return feed, nil
}

func (g *TemplateFeedGenerator) loadSource(context *tmpl.TemplateContext) (*url.URL, *Selection, error) {
	if err := g.config.Source.HTTP.Init(context); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize source: %w", err)
	}
	baseURLStr := g.config.Source.GetURL()
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, nil, err
	}
	context.Set("URL", baseURL.String())

	reader, err := g.config.Source.Open()
	if err != nil {
		return nil, nil, err
	}
	doc, err := newSelectionFromReader(reader)
	if err != nil {
		return nil, nil, err
	}
	context.Set("Content", doc)

	return baseURL, doc, nil
}

func (g *TemplateFeedGenerator) loadFeed(context *tmpl.TemplateContext, repository *repo.Repository) (*feeds.Feed, error) {
	feed := new(feeds.Feed)
	context.Set("Feed", feed)
	feed.Id = g.config.Feed.ID.MustEvaluate(context)
	if feedCache, err := repository.Feed.GetFeed(repo.IDKey(feed.Id)); err == nil {
		if feedCache == nil {
			feed.Title = g.config.Feed.Title.MustEvaluate(context)
			feed.Link = &feeds.Link{
				Href: g.config.Feed.Link.Href.MustEvaluate(context),
			}
			feed.Author = &feeds.Author{
				Name:  g.config.Feed.Author.Name.MustEvaluate(context),
				Email: g.config.Feed.Author.Email.MustEvaluate(context),
			}
			feed.Description = g.config.Feed.Description.MustEvaluate(context)
			feed.Created = time.Now()
			feed.Updated = feed.Created
		} else {
			feed = feedCache
			context.Set("Feed", feed)
		}
	} else {
		return nil, err
	}
	return feed, nil
}

func (g *TemplateFeedGenerator) loadItem(context *tmpl.TemplateContext, repository *repo.Repository, itemContent *Selection) (*feeds.Item, error) {
	context.Set("ItemContent", itemContent)
	context.Set("LinkContent", newSelectionFromFactory(func() (*goquery.Selection, error) {
		if g.config.Item.Link.HREF.IsDefined() {
			return loadDocument(g.config.Item.Link.HREF.MustEvaluate(context))
		}
		return nil, fmt.Errorf("'link' not defined in config file")
	}))

	// Evaluate 'id' first for getting cache.
	id := g.config.Item.ID.MustEvaluate(context)
	if len(id) == 0 {
		id = g.config.Item.Link.HREF.MustEvaluate(context)
		if len(id) == 0 {
			return nil, errors.New("'id' or 'link.href' is required")
		}
	}

	// Get cache or create new feed item
	key := repo.IDKey(id)
	if item, err := repository.Item.GetFeedItem(key); err == nil {
		if item == nil {
			item = new(feeds.Item)
			item.Id = id
			item.Title = g.config.Item.Title.MustEvaluate(context)
			item.Description = g.config.Item.Description.MustEvaluate(context)
			item.Author = &feeds.Author{
				Name: g.config.Item.Author.Name.MustEvaluate(context),
			}
			link := g.config.Item.Link.HREF.MustEvaluate(context)
			item.Content = g.config.Item.Content.MustEvaluate(context)
			item.Link = &feeds.Link{
				Href: link,
			}
			item.Created = time.Now()
			item.Updated = item.Created
			if err := repository.Item.PutFeedItem(key, item); err != nil {
				return nil, err
			}
		}
		return item, nil
	} else {
		return nil, err
	}
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
