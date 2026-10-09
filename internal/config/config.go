// Package config loads settings from the environment.
package config

import (
	"errors"
	"fmt"
)

type Config struct {
	DatabaseURL string
}

// Load reads the config through getenv (os.Getenv in main). Every setting is
// required; there are no defaults, so a missing variable fails loudly instead
// of silently pointing at the wrong database.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{DatabaseURL: getenv("DATABASE_URL")}

	var errs []error
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is not set"))
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
