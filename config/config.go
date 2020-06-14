package config

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Yamashou/gqlgenc/interseption"
	"github.com/vektah/gqlparser/v2/validator"

	"github.com/Yamashou/gqlgenc/client"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Model    config.PackageConfig `yaml:"model,omitempty"`
	Client   config.PackageConfig `yaml:"client,omitempty"`
	Models   config.TypeMap       `yaml:"models,omitempty"`
	Endpoint EndPointConfig       `yaml:"endpoint"`
	Query    []string             `yaml:"query"`

	GQLConfig *config.Config `yaml:"-"`
}

type EndPointConfig struct {
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

func findCfg(fileName string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "unable to get working dir to findCfg")
	}

	cfg := findCfgInDir(dir, fileName)

	if cfg == "" {
		return "", os.ErrNotExist
	}

	return cfg, nil
}

func findCfgInDir(dir, fileName string) string {
	path := filepath.Join(dir, fileName)
	return path
}

func LoadConfig(filename string) (*Config, error) {
	var cfg Config
	file, err := findCfg(filename)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get file path")
	}
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read config")
	}
	if err := yaml.UnmarshalStrict(b, &cfg); err != nil {
		return nil, errors.Wrap(err, "unable to parse config")
	}
	config.DefaultConfig()
	cfg.GQLConfig = &config.Config{
		Model:  cfg.Model,
		Models: cfg.Models,
		// TODO: gqlgen must be set exec but client not used
		Exec:       config.PackageConfig{Filename: "generated.go"},
		Directives: map[string]config.DirectiveConfig{},
	}

	if err := cfg.Client.Check(); err != nil {
		return nil, errors.Wrap(err, "config.exec")
	}

	return &cfg, nil
}

func (c *Config) LoadSchema(ctx context.Context) error {
	addHeader := func(req *http.Request) {
		for key, value := range c.Endpoint.Headers {
			req.Header.Set(key, value)
		}
	}
	gqlclient := client.NewClient(http.DefaultClient, c.Endpoint.URL, addHeader)

	schema, err := LoadRemoteSchema(ctx, gqlclient)
	if err != nil {
		return err
	}

	if schema.Query == nil {
		schema.Query = &ast.Definition{
			Kind: ast.Object,
			Name: "Query",
		}
		schema.Types["Query"] = schema.Query
	}

	c.GQLConfig.Schema = schema
	return nil
}

func LoadRemoteSchema(ctx context.Context, gqlclient *client.Client) (*ast.Schema, error) {
	var res interseption.IntrospectionQuery
	if err := gqlclient.Post(ctx, interseption.Introspection, &res, nil); err != nil {
		fmt.Println(err)
		return nil, err
	}

	var doc ast.SchemaDocument
	typeMap := make(map[string]*interseption.FullType)
	for _, typ := range res.Schema.Types {
		typeMap[*typ.Name] = typ
	}
	for _, typeVale := range typeMap {
		doc.Definitions = append(doc.Definitions, interseption.ParseTypeSystemDefinition(typeVale))
	}

	for _, directiveValue := range res.Schema.Directives {
		doc.Directives = append(doc.Directives, interseption.ParseDirectiveDefinition(directiveValue))
	}

	schema, err := validator.ValidateSchemaDocument(&doc)
	if err != nil {
		return nil, err
	}

	return schema, nil
}