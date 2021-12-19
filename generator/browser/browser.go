package browser

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gorilla/feeds"
	"github.com/uphy/feedgen/config"
	"github.com/uphy/feedgen/generator"
	"github.com/uphy/feedgen/template"
)

type (
	BrowserFeedGeneratorConfig struct {
		URL     template.TemplateField `yaml:"url"`
		Actions []ActionConfig         `yaml:"actions"`
		Browser struct {
			Visible bool           `yaml:"visible"`
			Timeout *time.Duration `yaml:"timeout"`
		} `yaml:"browser"`
	}
	BrowserFeedGenerator struct {
		noSandbox bool
		config    *BrowserFeedGeneratorConfig
	}
	ActionConfig struct {
		WaitVisible *template.TemplateField `yaml:"waitVisible"`
		Feed        *template.TemplateField `yaml:"feed"`
		Items       *template.TemplateField `yaml:"items"`
		Sleep       *time.Duration          `yaml:"sleep"`
	}
)

func New(noSandbox bool) *BrowserFeedGenerator {
	return &BrowserFeedGenerator{noSandbox: noSandbox}
}

func (g *BrowserFeedGenerator) LoadOptions(options config.GeneratorOptions) error {
	var config *BrowserFeedGeneratorConfig
	if err := options.Unmarshal(&config); err != nil {
		return err
	}
	g.config = config
	return nil
}

func (g *BrowserFeedGenerator) Generate(generatorContext *generator.Context) (*feeds.Feed, error) {
	templateContext := generatorContext.TemplateContext
	url := g.config.URL.MustEvaluate(templateContext)

	// Start Chrome
	ctx, cancel := g.buildChromeContext()
	defer cancel()

	feed := new(feeds.Feed)
	feed.Link = &feeds.Link{
		Href: url,
	}

	// Build actions
	actions := make([]chromedp.Action, 0)
	actions = append(actions, chromedp.Navigate(url))
	for _, command := range g.config.Actions {
		if command.WaitVisible != nil {
			query := command.WaitVisible.MustEvaluate(templateContext)
			actions = append(actions, chromedp.WaitVisible(query, chromedp.ByQuery))
		} else if command.Feed != nil {
			script := command.Feed.MustEvaluate(templateContext)
			actions = append(actions, chromedp.Evaluate(script, &feed))
		} else if command.Items != nil {
			script := command.Items.MustEvaluate(templateContext)
			actions = append(actions, chromedp.Evaluate(script, &feed.Items))
		} else if command.Sleep != nil {
			actions = append(actions, chromedp.Sleep(*command.Sleep))
		}
	}
	// Run actions
	err := chromedp.Run(ctx, actions...)
	if err != nil {
		return nil, fmt.Errorf("failed on Chrome action: %w", err)
	}
	return feed, nil
}

func (g *BrowserFeedGenerator) buildChromeContext() (context.Context, func()) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", !g.config.Browser.Visible),
		chromedp.Flag("window-size", "1920,1080"),
	)
	if g.noSandbox {
		// for execution on heroku
		opts = append(opts,
			chromedp.Flag("no-sandbox", "true"),
			chromedp.Flag("disable-setuid-sandbox", "true"),
		)
	}
	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	if g.config.Browser.Timeout != nil {
		ctx, cancel = context.WithTimeout(ctx, *g.config.Browser.Timeout)

	}
	return ctx, cancel
}
