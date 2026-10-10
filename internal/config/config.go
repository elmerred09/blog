// Package config loads settings from the environment.
package config

import (
	"errors"
	"fmt"
)

// DefaultHTTPAddr is used when HTTP_ADDR is unset.
const DefaultHTTPAddr = ":8080"

type Config struct {
	DatabaseURL string
	HTTPAddr    string
}

// Load reads the config through getenv (os.Getenv in main).
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		DatabaseURL: getenv("DATABASE_URL"),
		HTTPAddr:    getenv("HTTP_ADDR"),
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = DefaultHTTPAddr
	}

	var errs []error
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is not set"))
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
