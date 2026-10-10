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

	tests := []struct {
		name string
		env  map[string]string
		want config.Config
	}{
		{
			name: "HTTP_ADDR defaults",
			env:  map[string]string{"DATABASE_URL": "postgres://x/y"},
			want: config.Config{DatabaseURL: "postgres://x/y", HTTPAddr: config.DefaultHTTPAddr},
		},
		{
			name: "everything set",
			env:  map[string]string{"DATABASE_URL": "postgres://x/y", "HTTP_ADDR": "127.0.0.1:9000"},
			want: config.Config{DatabaseURL: "postgres://x/y", HTTPAddr: "127.0.0.1:9000"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := config.Load(env(tc.env))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got != tc.want {
				t.Errorf("Load() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Parallel()

	_, err := config.Load(env(map[string]string{"HTTP_ADDR": ":9000"}))
	if err == nil {
		t.Fatal("Load succeeded without DATABASE_URL")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error %q does not name DATABASE_URL", err)
	}
}
