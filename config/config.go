package config

import (
	"errors"
	"os"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

const defaultConfigFile = "config.yml"

type Config struct {
	APIKey     string
	Version    string
	Categories []string               `yaml:"categories"`
	Filters    map[string]interface{} `yaml:"filters"`
}

func NewConfig(path string) (Config, error) {
	if path == "" {
		path = defaultConfigFile
	}

	var cfg Config
	buffer, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(buffer, &cfg); err != nil {
			return Config{}, xerrors.Errorf(": %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Config{}, xerrors.Errorf(": %w", err)
	}

	cfg.APIKey = os.Getenv("README_API_KEY")
	cfg.Version = os.Getenv("README_VERSION")

	return cfg, nil
}
