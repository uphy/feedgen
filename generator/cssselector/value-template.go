package cssselector

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"

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
		Fields      map[string]*Field
	}
	fieldYaml struct {
		Selector string `yaml:"selector"`
		Template string `yaml:"template"`
		Attr     string `yaml:"attr"`
	}
)

func newContext(url *url.URL, doc *Selection) *TemplateContext {
	return &TemplateContext{
		URL:         url,
		Document:    doc,
		ItemContent: nil,
		Fields:      make(map[string]*Field),
	}
}

func (c *TemplateContext) addField(name string, t *Field) error {
	if t != nil {
		c.Fields[name] = t
		if err := t.init(c, name); err != nil {
			return err
		}
	}
	return nil
}

func (c *TemplateContext) prepare(itemContent *Selection) {
	c.ItemContent = itemContent
	c.LinkContent = newSelectionFromFactory(func() (*goquery.Selection, error) {
		if link, exist := c.Fields["link"]; exist {
			return loadDocument(link.String())
		}
		return nil, fmt.Errorf("'link' not defined in config file")
	})
	for _, f := range c.Fields {
		f.clearCache()
	}
}

func newTemplateFieldValue(template string) *templateFieldValue {
	return &templateFieldValue{rawTemplate: template}
}

func (f *templateFieldValue) init(context *TemplateContext) error {
	if parsed, err := template.New("tmpl").Funcs(template.FuncMap{
		"get": func(name string) (*Field, error) {
			if field, exist := context.Fields[name]; exist {
				return field, nil
			} else {
				return nil, nil
			}
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
