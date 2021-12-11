package config

import (
	"fmt"
	"os"

	"github.com/uphy/feedgen/template"
	"gopkg.in/yaml.v2"
)

type (
	Config struct {
		Include    []string                    `yaml:"include"`
		Generators map[string]*GeneratorConfig `yaml:"generators"`
	}
	GeneratorConfig struct {
		Endpoint template.TemplateField

		Type    string
		Options GeneratorOptions
	}
	GeneratorOptions map[string]interface{}
)

func ParseConfig(file string) (*Config, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var c Config
	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func ParseGeneratorConfig(b []byte) (*GeneratorConfig, error) {
	var c GeneratorConfig
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *GeneratorOptions) Unmarshal(i interface{}) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, i)
}

func (c *GeneratorConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var m map[string]interface{}
	if err := unmarshal(&m); err != nil {
		return err
	}

	if t, ok := m["type"].(string); ok {
		c.Type = t
		delete(m, "type")
	} else {
		return fmt.Errorf("'type' must be a string: %v", m)
	}

	if e, exist := m["endpoint"]; exist {
		if endpoint, ok := e.(string); ok {
			c.Endpoint = template.NewTemplateField(endpoint)
			delete(m, "endpoint")
		} else {
			return fmt.Errorf("'endpoint' must be a string: %v", m)
		}
	} else {
		return fmt.Errorf("'endpoint' is required")
	}

	c.Options = m
	return nil
}
