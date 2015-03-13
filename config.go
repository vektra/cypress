package cypress

import (
	"io"
	"io/ioutil"
	"time"

	"github.com/naoina/toml"
)

type Config struct {
	S3 struct {
		AllowUnsigned bool
		SignKey       string
	}
}

func ParseConfig(r io.Reader) (*Config, error) {
	var cfg Config

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = toml.Unmarshal(data, &cfg)
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

	return toml.Unmarshal(data, &cfg)
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalTOML(data []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(data[1 : len(data)-1]))
	return err
}
