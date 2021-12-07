package cssselector

type (
	constantFieldValue struct {
		value string
	}
)

func newConstantFieldValue(value string) *constantFieldValue {
	return &constantFieldValue{value}
}

func (f *constantFieldValue) init(context *TemplateContext) error {
	return nil
}

func (f *constantFieldValue) get() (string, error) {
	return f.value, nil
}
