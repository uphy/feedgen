package cssselector

import (
	"fmt"
	"strings"
)

type (
	Field struct {
		value        FieldValue
		ResultMapper func(value string) (string, error)
		// on evaluation
		resultCache *string
	}
	FieldValue interface {
		init(templateContext *TemplateContext) error
		get() (string, error)
	}
	fieldYaml struct {
		Selector string `yaml:"selector"`
		Attr     string `yaml:"attr"`
		Template string `yaml:"template"`
		Constant string `yaml:"constant"`
	}
)

func (f *Field) init(context *TemplateContext) error {
	f.resultCache = nil
	if f.IsDefined() {
		if err := f.value.init(context); err != nil {
			return fmt.Errorf("failed to init value: err=%w", err)
		}
		return nil
	} else {
		return nil
	}
}

// IsDefined returns true if the user config defined this field.
func (f *Field) IsDefined() bool {
	return f.value != nil
}

func (f *Field) Eval() (string, error) {
	if f.resultCache != nil {
		return *f.resultCache, nil
	}
	if !f.IsDefined() {
		return "", nil
	}
	result, err := f.value.get()
	if err != nil {
		return "", err
	}
	result = trimSpace(result)
	if f.ResultMapper != nil {
		if r, err := f.ResultMapper(result); err == nil {
			result = r
		} else {
			return "", fmt.Errorf("failed to map template result value: err=%w", err)
		}
	}
	f.resultCache = &result
	return result, nil
}

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

func (f *Field) String() string {
	if result, err := f.Eval(); err == nil {
		return result
	} else {
		panic(err)
	}
}

func (f *Field) clearCache() {
	f.resultCache = nil
}

func (f *Field) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var fieldYaml fieldYaml
	if err := unmarshal(&fieldYaml); err == nil {
		if fieldYaml.Selector != "" && fieldYaml.Template != "" {
			return fmt.Errorf("cannot use both 'selector' and 'template'")
		}
		if fieldYaml.Template != "" {
			if fieldYaml.Attr != "" {
				return fmt.Errorf("cannot use both 'template' and 'attr'")
			}
			f.value = newTemplateFieldValue(fieldYaml.Template)
			return nil
		}
		if fieldYaml.Constant != "" {
			f.value = newConstantFieldValue(fieldYaml.Constant)
			return nil
		}
		f.value = newSelectorFieldValue(fieldYaml.Selector, fieldYaml.Attr)
		return nil
	}

	var selector string
	if err := unmarshal(&selector); err != nil {
		return err
	}
	f.value = newSelectorFieldValue(selector, "")
	return nil
}
