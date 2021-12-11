package converter

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"

	"github.com/gorilla/feeds"
)

//go:embed template.html
var htmlTemplate string

type htmlConverter struct {
}

func (c *htmlConverter) Convert(feed *feeds.Feed) (*Result, error) {
	tmpl, err := template.New("converter-html").Funcs(template.FuncMap{
		"html": func(value interface{}) template.HTML {
			return template.HTML(fmt.Sprint(value))
		},
	}).Parse(htmlTemplate)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, map[string]interface{}{"Feed": feed}); err != nil {
		return nil, err
	}
	return newResult("text/html", buf.String()), nil
}
