package source

import (
	"io"
	"net/http"

	"github.com/uphy/feedgen/template"
)

type (
	httpSourceHandler struct {
		URL template.TemplateField `yaml:"url"`

		context *template.TemplateContext
		url     string
	}
	httpSourceConfigYAML struct {
		URL template.TemplateField `yaml:"url"`
	}
)

func (c *httpSourceHandler) Init(context *template.TemplateContext) error {
	c.context = context
	if s, err := c.URL.Evaluate(context); err == nil {
		c.url = s
	} else {
		return err
	}
	return nil
}

func (c *httpSourceHandler) GetURL() string {
	return c.URL.MustEvaluate(c.context)
}

func (c *httpSourceHandler) Open() (io.ReadCloser, error) {
	req, err := http.Get(c.url)
	if err != nil {
		return nil, err
	}
	return req.Body, nil
}

func (c *httpSourceHandler) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var y httpSourceConfigYAML
	if err := unmarshal(&y); err == nil {
		c.URL = y.URL
	} else {
		var s string
		if err := unmarshal(&s); err == nil {
			c.URL = template.NewTemplateField(s)
		} else {
			return err
		}
	}
	return nil
}
