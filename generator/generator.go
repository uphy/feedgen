package generator

import (
	"embed"
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"

	"github.com/uphy/feedgen/config"
	"github.com/uphy/feedgen/repo"
	"github.com/uphy/feedgen/template"

	"github.com/gorilla/feeds"
)

type (
	Context struct {
		Repository      *repo.Repository
		TemplateContext *template.TemplateContext
	}

	FeedGenerator interface {
		LoadOptions(options config.GeneratorOptions) error
		Generate(context *Context) (*feeds.Feed, error)
	}

	FeedGeneratorWrapper struct {
		Name      string
		Endpoint  string
		generator FeedGenerator
	}

	FeedGenerators struct {
		registry        map[string]func() FeedGenerator
		Generators      map[string]*FeedGeneratorWrapper
		repository      *repo.Repository
		templateContext *template.TemplateContext
	}
)

//go:embed config
var predefinedGeneratorConfigs embed.FS

func New(repository *repo.Repository) *FeedGenerators {
	f := &FeedGenerators{
		registry:        make(map[string]func() FeedGenerator),
		Generators:      make(map[string]*FeedGeneratorWrapper),
		repository:      repository,
		templateContext: template.NewRootTemplateContext(),
	}
	return f
}

func findPreDefinedGeneratorConfig(name string) (*config.GeneratorConfig, error) {
	b, err := predefinedGeneratorConfigs.ReadFile(filepath.Join("config", name+".yml"))
	if err != nil {
		return nil, err
	}
	return config.ParseGeneratorConfig(b)
}

func (f *FeedGenerators) Register(name string, v interface{}) {
	t := reflect.TypeOf(v)
	f.registry[name] = func() FeedGenerator {
		v := reflect.New(t).Interface()
		return v.(FeedGenerator)
	}
}

func (f *FeedGenerators) newGenerator(c *config.GeneratorConfig) (FeedGenerator, error) {
	factory, exist := f.registry[c.Type]
	if !exist {
		return nil, fmt.Errorf("unknown generator type: %s", c.Type)
	}
	gen := factory()
	if err := gen.LoadOptions(c.Options); err != nil {
		return nil, err
	}
	return gen, nil
}

func (f *FeedGenerators) LoadConfig(config *config.Config) error {
	for k := range f.Generators {
		delete(f.Generators, k)
	}

	for _, generatorName := range config.Include {
		generatorConfig, err := findPreDefinedGeneratorConfig(generatorName)
		if err != nil {
			return fmt.Errorf("failed to include a generator config: name=%s, err=%w", generatorName, err)
		}
		if err := f.loadGeneratorConfig(generatorName, generatorConfig); err != nil {
			return fmt.Errorf("failed to load included generator config: name=%s, err=%w", generatorName, err)
		}
	}

	for generatorName, generatorConfig := range config.Generators {
		f.loadGeneratorConfig(generatorName, generatorConfig)
	}

	return nil
}

func (f *FeedGenerators) loadGeneratorConfig(generatorName string, generatorConfig *config.GeneratorConfig) error {
	gen, err := f.newGenerator(generatorConfig)
	if err != nil {
		return fmt.Errorf("failed to load '%s': %w", generatorName, err)
	}
	endpoint, err := generatorConfig.Endpoint.Evaluate(f.templateContext)
	if err != nil {
		return fmt.Errorf("failed to evaluate 'endpoint': endpoint=%v, err=%w", generatorConfig.Endpoint, err)
	}
	f.Generators[generatorName] = &FeedGeneratorWrapper{generatorName, endpoint, gen}
	return nil
}

func (f *FeedGenerators) Generate(name string, parameters map[string]string, queryParameters url.Values) (*feeds.Feed, error) {
	wrapper, ok := f.Generators[name]
	if !ok {
		return nil, fmt.Errorf("generator not found: %s", name)
	}
	gen := wrapper.generator

	context := &Context{f.repository, f.templateContext.Child()}
	context.TemplateContext.Set("Parameters", parameters)
	context.TemplateContext.Set("QueryParameters", queryParameters)
	context.TemplateContext.AddFuncs(map[string]interface{}{
		"Param": func(name string) *string {
			if value, exist := parameters[name]; exist {
				return &value
			} else {
				return nil
			}
		},
		"QueryParam": func(name string) *string {
			if queryParameters.Has(name) {
				value := queryParameters.Get(name)
				return &value
			} else {
				return nil
			}
		},
		"QueryParams": func() *string {
			p := queryParameters.Encode()
			return &p
		},
	})
	if feed, err := gen.Generate(context); err == nil {
		return feed, nil
	} else {
		return nil, err
	}
}
