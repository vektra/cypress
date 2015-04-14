package cypress

import (
	"io"
	"io/ioutil"
	"time"

	"github.com/naoina/toml"
	"github.com/naoina/toml/ast"
)

type Config struct {
	trees []*ast.Table
}

func ParseConfig(r io.Reader) (*Config, error) {
	var cfg Config

	err := cfg.Add(r)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func LoadMergedConfig(path string, cfg *Config) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	tree, err := toml.Parse(data)
	if err != nil {
		return err
	}

	cfg.trees = append(cfg.trees, tree)

	return nil
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalTOML(data []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(data[1 : len(data)-1]))
	return err
}

func subTable(name string, t *ast.Table) (*ast.Table, bool) {
	sv, ok := t.Fields[name]
	if !ok {
		return nil, false
	}

	sub, ok := sv.(*ast.Table)
	if !ok {
		return nil, false
	}

	return sub, true
}

func (cfg *Config) Add(r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	tree, err := toml.Parse(data)
	if err != nil {
		return err
	}

	cfg.trees = append(cfg.trees, tree)

	return nil
}

func (cfg *Config) AddString(s string) error {
	tree, err := toml.Parse([]byte(s))
	if err != nil {
		return err
	}

	cfg.trees = append(cfg.trees, tree)

	return nil
}

func (cfg *Config) Load(name string, v interface{}) error {
	for _, tree := range cfg.trees {
		if sub, ok := subTable(name, tree); ok {
			err := toml.UnmarshalTable(sub, v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
