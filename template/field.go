package template

type (
	TemplateField struct {
		template     string
		defined      bool
		ResultMapper func(value string) (string, error)
	}
)

func NewTemplateField(template string) TemplateField {
	return TemplateField{template, true, nil}
}

func (t TemplateField) MustEvaluate(ctx *TemplateContext) string {
	if result, err := t.Evaluate(ctx); err == nil {
		return result
	} else {
		panic(err)
	}
}

func (t TemplateField) Evaluate(ctx *TemplateContext) (string, error) {
	if evaluated, err := evaluate(t.template, ctx); err == nil {
		if t.ResultMapper != nil {
			if s, err := t.ResultMapper(evaluated); err == nil {
				evaluated = s
			} else {
				return "", err
			}
		}
		return evaluated, nil
	} else {
		return "", err
	}
}

func (t TemplateField) IsDefined() bool {
	return t.defined
}

func (t *TemplateField) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var template string
	if err := unmarshal(&template); err != nil {
		return err
	}
	t.template = template
	t.defined = true
	return nil
}
