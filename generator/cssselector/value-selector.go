package cssselector

import "fmt"

type (
	selectorFieldValue struct {
		selector string
		attr     string

		context *TemplateContext
	}
)

func newSelectorFieldValue(selector, attr string) *selectorFieldValue {
	return &selectorFieldValue{selector: selector, attr: attr}
}

func (f *selectorFieldValue) init(context *TemplateContext) error {
	f.context = context
	return nil
}

func (f *selectorFieldValue) get() (string, error) {
	if s := f.context.ItemContent.Select(f.selector); s.Exist() {
		attr := s.Attr(f.attr)
		if attr == "" {
			return "", fmt.Errorf("attr not found: selector=%s, attr=%s", f.selector, f.attr)
		}
		return attr, nil
	} else {
		return "", fmt.Errorf("element not found: selector=%s, attr=%s", f.selector, f.attr)
	}
}
