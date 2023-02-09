package config

import (
	"os"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

const defaultConfigFile = ".readme-sync-config.yml"

type CategoryConfig struct {
	Slug  string `yaml:"slug"`
	Title string `yaml:"title"`
}

type Config struct {
	Categories []CategoryConfig `yaml:"categories"`
	Version    string           `yaml:"version"`
	Key        string           `yaml:"-"`
}

func NewConfig(path string) (Config, error) {
	if path == "" {
		path = defaultConfigFile
	}

	cfgFile, err := os.Open(path)
	if err != nil {
		return Config{}, xerrors.Errorf(": %w", err)
	}

	var cfg Config
	if err := yaml.NewDecoder(cfgFile).Decode(&cfg); err != nil {
		return Config{}, xerrors.Errorf(": %w", err)
	}

	// only source key from environment to prevent users from accidentally misplacing keys
	key, ok := os.LookupEnv("README_APIKEY")
	if !ok {
		return Config{}, xerrors.New("README_APIKEY not found in environment")
	}
	if key == "" {
		return Config{}, xerrors.New("README_APIKEY cannot be an empty value")
	}
	cfg.Key = key

	return cfg, nil
}
