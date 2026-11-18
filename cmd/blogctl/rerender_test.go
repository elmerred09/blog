package main

import (
	"strings"
	"testing"
)

func noEnv(string) string { return "" }

func TestRunRerenderErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		getenv  func(string) string
		wantErr string
	}{
		{"without DATABASE_URL", []string{"rerender"}, noEnv, "DATABASE_URL is not set"},
		{"with arguments", []string{"rerender", "extra"}, noEnv, "takes no arguments"},
		{
			"with a malformed DATABASE_URL",
			[]string{"rerender"},
			func(string) string { return "postgres://%zz" },
			"database config",
		},
		{"unknown command", []string{"rerendr"}, noEnv, `unknown command "rerendr"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out strings.Builder
			err := run(t.Context(), tc.args, tc.getenv, &out)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want one containing %q", err, tc.wantErr)
			}
			if out.Len() != 0 {
				t.Errorf("wrote output despite the error:\n%s", out.String())
			}
		})
	}
}
