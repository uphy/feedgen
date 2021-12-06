package cssselector

import (
	"fmt"
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
	var templateYaml fieldYaml
	if err := unmarshal(&templateYaml); err == nil {
		if templateYaml.Selector != "" && templateYaml.Template != "" {
			return fmt.Errorf("cannot use both 'selector' and 'template'")
		}
		if templateYaml.Template != "" {
			if templateYaml.Attr != "" {
				return fmt.Errorf("cannot use both 'template' and 'attr'")
			}
			f.value = newTemplateFieldValue(templateYaml.Template)
			return nil
		}
		f.value = newSelectorFieldValue(templateYaml.Selector, templateYaml.Attr)
		return nil
	}

	var selector string
	if err := unmarshal(&selector); err != nil {
		return err
	}
	f.value = newSelectorFieldValue(selector, "")
	return nil
}
