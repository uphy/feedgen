package cssselector

import (
	"bytes"
	"fmt"
	"net/url"
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
		Document    *Selection
		ItemContent *Selection
		LinkContent *Selection
		Fields      *ItemConfig
		allFields   []*Field
	}
	fieldYaml struct {
		Selector string `yaml:"selector"`
		Template string `yaml:"template"`
		Attr     string `yaml:"attr"`
	}
)

func newContext(url *url.URL, doc *Selection, fields *ItemConfig) (*TemplateContext, error) {
	ctx := &TemplateContext{
		URL:         url,
		Document:    doc,
		ItemContent: nil,
		Fields:      fields,
		allFields: []*Field{
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
	if parsed, err := template.New("tmpl").Parse(f.rawTemplate); err != nil {
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
