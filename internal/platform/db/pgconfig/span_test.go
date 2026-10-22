package pgconfig

import "testing"

func TestSQLCSpanName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		stmt string
		want string
	}{
		{"sqlc", "-- name: GetPostBySlug :one\nSELECT 1", "GetPostBySlug"},
		{"no comment", "SELECT 1", "query"},
		{"empty name", "-- name: ", "query"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := sqlcSpanName(tc.stmt); got != tc.want {
				t.Errorf("sqlcSpanName() = %q, want %q", got, tc.want)
			}
		})
	}
}
