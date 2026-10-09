package config_test

import (
	"strings"
	"testing"

	"github.com/elmerred09/blog/internal/config"
)

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func TestLoad(t *testing.T) {
	t.Parallel()

	got, err := config.Load(env(map[string]string{"DATABASE_URL": "postgres://x/y"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := config.Config{DatabaseURL: "postgres://x/y"}
	if got != want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Parallel()

	_, err := config.Load(env(nil))
	if err == nil {
		t.Fatal("Load succeeded without DATABASE_URL")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error %q does not name DATABASE_URL", err)
	}
}
