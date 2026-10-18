package media

import (
	"errors"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/elmerred09/blog/internal/validation"
)

func testUploader(t *testing.T) *Uploader {
	t.Helper()
	u, err := New(Config{
		Endpoint:      "127.0.0.1:9",
		Region:        "us-east-1",
		Bucket:        "blog",
		AccessKey:     "blog",
		SecretKey:     "blogsecret",
		PublicBaseURL: "http://127.0.0.1:9000/blog/",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return u
}

func TestPresignImageUpload(t *testing.T) {
	t.Parallel()
	u := testUploader(t)

	got, err := u.PresignImageUpload(t.Context(), "image/png", 4)
	if err != nil {
		t.Fatalf("PresignImageUpload: %v", err)
	}

	parsed, err := url.Parse(got.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	key := strings.TrimPrefix(parsed.Path, "/blog/")
	if !strings.HasPrefix(key, "images/") || !strings.HasSuffix(key, ".png") {
		t.Errorf("key = %q, want images/<id>.png", key)
	}
	if got.PublicURL != "http://127.0.0.1:9000/blog/"+key {
		t.Errorf("public url = %q, want it to be the same key under the public base", got.PublicURL)
	}
	if got.Headers["Content-Type"] != "image/png" || got.Headers["Content-Length"] != "4" {
		t.Errorf("headers = %v, want Content-Type image/png and Content-Length 4", got.Headers)
	}

	signed := strings.Split(parsed.Query().Get("X-Amz-SignedHeaders"), ";")
	for _, want := range []string{"content-length", "content-type"} {
		if !slices.Contains(signed, want) {
			t.Errorf("X-Amz-SignedHeaders = %q, want it to contain %q", signed, want)
		}
	}
	if parsed.Query().Get("X-Amz-Expires") != "300" {
		t.Errorf("X-Amz-Expires = %q, want 300", parsed.Query().Get("X-Amz-Expires"))
	}

	again, err := u.PresignImageUpload(t.Context(), "image/png", 4)
	if err != nil {
		t.Fatalf("second presign: %v", err)
	}
	if again.URL == got.URL {
		t.Error("two uploads got the same URL; the key must be generated, not chosen by the caller")
	}
}

func TestPresignImageUploadRejects(t *testing.T) {
	t.Parallel()
	u := testUploader(t)

	tests := []struct {
		name        string
		contentType string
		size        int64
		fields      []string
	}{
		{"svg", "image/svg+xml", 4, []string{"content_type"}},
		{"parameters on the type", "image/png; charset=binary", 4, []string{"content_type"}},
		{"empty type", "", 4, []string{"content_type"}},
		{"zero bytes", "image/gif", 0, []string{"size"}},
		{"negative", "image/webp", -1, []string{"size"}},
		{"one past the limit", "image/jpeg", MaxBytes + 1, []string{"size"}},
		{"both", "image/svg+xml", 0, []string{"content_type", "size"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := u.PresignImageUpload(t.Context(), tc.contentType, tc.size)
			var invalid *validation.Error
			if !errors.As(err, &invalid) {
				t.Fatalf("err = %v, want a *validation.Error", err)
			}
			got := make([]string, len(invalid.Problems))
			for i, p := range invalid.Problems {
				got[i] = p.Field
				if p.Message == "" {
					t.Errorf("problem %s has an empty message", p.Field)
				}
			}
			if !slices.Equal(got, tc.fields) {
				t.Errorf("fields = %v, want %v", got, tc.fields)
			}
		})
	}

	t.Run("the limit itself is accepted", func(t *testing.T) {
		t.Parallel()
		for _, ct := range []string{"image/png", "image/jpeg", "image/webp", "image/gif"} {
			got, err := u.PresignImageUpload(t.Context(), ct, MaxBytes)
			if err != nil {
				t.Fatalf("%s: %v", ct, err)
			}
			if !strings.Contains(got.PublicURL, "images/") {
				t.Errorf("%s public url = %q", ct, got.PublicURL)
			}
		}
	})
}

func TestNewRequiresConfig(t *testing.T) {
	t.Parallel()
	if _, err := New(Config{}); err == nil {
		t.Fatal("New with an empty config succeeded")
	}
}
