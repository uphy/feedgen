package template

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"text/template"

	"github.com/PuerkitoBio/goquery"
)

type (
	templateFieldValue struct {
		// required
		rawTemplate string

		// on init()
		parsedTemplate  *template.Template
		templateContext *TemplateContext
	}
	TemplateContext struct {
		URL         *url.URL
		Content     *Selection
		ItemContent *Selection
		LinkContent *Selection
		Fields      *ItemConfig
		allFields   []*Field
	}
)

func newContext(url *url.URL, content *Selection, feed *FeedConfig, fields *ItemConfig) (*TemplateContext, error) {
	ctx := &TemplateContext{
		URL:         url,
		Content:     content,
		ItemContent: nil,
		Fields:      fields,
		allFields: []*Field{
			&feed.ID,
			&feed.Title,
			&feed.Link.Href,
			&feed.Description,
			&feed.Author.Name,
			&feed.Author.Email,
			&fields.Author,
			&fields.Content,
			&fields.Description,
			&fields.ID,
			&fields.Title,
			&fields.Link.HREF,
		},
	}
	for _, f := range ctx.allFields {
		if err := f.init(ctx); err != nil {
			return nil, err
		}
	}
	return ctx, nil
}

func (c *TemplateContext) prepare(itemContent *Selection) {
	c.ItemContent = itemContent
	c.LinkContent = newSelectionFromFactory(func() (*goquery.Selection, error) {
		if c.Fields.Link.HREF.IsDefined() {
			return loadDocument(c.Fields.Link.HREF.String())
		}
		return nil, fmt.Errorf("'link' not defined in config file")
	})
	for _, f := range c.allFields {
		f.clearCache()
	}
}

func newTemplateFieldValue(template string) *templateFieldValue {
	return &templateFieldValue{rawTemplate: template}
}

func (f *templateFieldValue) init(context *TemplateContext) error {
	if parsed, err := template.New("tmpl").Funcs(template.FuncMap{
		"ReplaceAll": func(old, new, s string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"TrimSpace": strings.TrimSpace,
		"Attr": func(attr string, input *Selection) string {
			return input.Attr(attr)
		},
		"Text": func(input *Selection) string {
			return input.Text()
		},
	}).Parse(f.rawTemplate); err != nil {
		return fmt.Errorf("failed to parse template: err=%w", err)
	} else {
		f.parsedTemplate = parsed
		f.templateContext = context
	}
	return nil
}

func (f *templateFieldValue) get() (string, error) {
	buf := new(bytes.Buffer)
	if err := f.parsedTemplate.Execute(buf, f.templateContext); err != nil {
		return "", fmt.Errorf("failed to evaluate template: err=%w", err)
	}
	return buf.String(), nil
}
