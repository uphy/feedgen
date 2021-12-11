package source

import (
	"io"

	"github.com/uphy/feedgen/template"
)

type (
	Source struct {
		HTTP *httpSourceHandler `yaml:"http"`
	}
	sourceHandler interface {
		Init(context *template.TemplateContext) error
		GetURL() string
		Open() (io.ReadCloser, error)
	}
)

func (s *Source) source() sourceHandler {
	if s.HTTP != nil {
		return s.HTTP
	}
	panic("invalid source")
}

func (s *Source) GetURL() string {
	return s.source().GetURL()
}

func (s *Source) Open() (io.ReadCloser, error) {
	return s.source().Open()
}
