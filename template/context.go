package template

type (
	TemplateContext struct {
		parent    *TemplateContext
		variables map[string]interface{}
		funcs     map[string]interface{}
	}
)

func NewRootTemplateContext() *TemplateContext {
	return newTemplateContext(nil)
}

func newTemplateContext(parent *TemplateContext) *TemplateContext {
	return &TemplateContext{parent, make(map[string]interface{}), make(map[string]interface{})}
}

func (c *TemplateContext) Child() *TemplateContext {
	return newTemplateContext(c)
}

func (c *TemplateContext) Set(key string, value interface{}) {
	c.variables[key] = value
}

func (c *TemplateContext) AddFuncs(funcs map[string]interface{}) {
	for k, v := range funcs {
		if _, exist := c.funcs[k]; exist {
			panic("already exist: " + k)
		}
		c.funcs[k] = v
	}
}

func (c *TemplateContext) flatten() (vars map[string]interface{}, funcs map[string]interface{}) {
	if c.parent != nil {
		vars = make(map[string]interface{})
		funcs = make(map[string]interface{})
		parentVars, parentFuncs := c.parent.flatten()
		for k, v := range parentVars {
			vars[k] = v
		}
		for k, v := range parentFuncs {
			funcs[k] = v
		}
		for k, v := range c.variables {
			vars[k] = v
		}
		for k, v := range c.funcs {
			funcs[k] = v
		}
		return
	}
	return c.variables, c.funcs
}
