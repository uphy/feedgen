package generator

import (
	"fmt"
	"reflect"

	"github.com/uphy/feedgen/config"
	"github.com/uphy/feedgen/repo"

	"github.com/gorilla/feeds"
)

type (
	Context struct {
		Repository *repo.Repository
	}

	FeedGenerator interface {
		LoadOptions(options config.GeneratorOptions) error
		Generate(feed *feeds.Feed, context *Context) error
	}

	feedGeneratorWrapper struct {
		generator FeedGenerator
	}

	FeedGenerators struct {
		registry   map[string]func() FeedGenerator
		generators map[string]*feedGeneratorWrapper
		context    *Context
	}
)

func New(repository *repo.Repository) *FeedGenerators {
	f := &FeedGenerators{
		registry:   make(map[string]func() FeedGenerator),
		generators: make(map[string]*feedGeneratorWrapper),
	}
	f.context = &Context{repository}
	return f
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
	for k := range f.generators {
		delete(f.generators, k)
	}

	for k, v := range config.Generators {
		gen, err := f.newGenerator(v)
		if err != nil {
			return fmt.Errorf("failed to load '%s': %w", k, err)
		}
		f.generators[k] = &feedGeneratorWrapper{gen}
	}
	return nil
}

func (f *FeedGenerators) Generate(name string) (*feeds.Feed, error) {
	wrapper, ok := f.generators[name]
	if !ok {
		return nil, fmt.Errorf("generator not found: %s", name)
	}
	gen := wrapper.generator
	feed := new(feeds.Feed)

	if err := gen.Generate(feed, f.context); err != nil {
		return nil, err
	}
	return feed, nil
}
