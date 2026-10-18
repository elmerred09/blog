package config_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
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

func TestLoadAdminIsOptional(t *testing.T) {
	t.Parallel()
	got, err := config.Load(env(map[string]string{"DATABASE_URL": "postgres://x/y"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Admin != nil {
		t.Errorf("Admin = %+v, want nil when no admin variable is set", got.Admin)
	}
}

func TestLoadAdmin(t *testing.T) {
	t.Parallel()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	escaped := strings.ReplaceAll(string(pemBytes), "\n", `\n`)

	got, err := config.Load(env(map[string]string{
		"DATABASE_URL":         "postgres://x/y",
		"ADMIN_JWT_PUBLIC_KEY": escaped,
		"ADMIN_SUBJECTS":       "github:1, github:2",
		"S3_ENDPOINT":          "https://acct.r2.cloudflarestorage.com",
		"S3_REGION":            "auto",
		"S3_BUCKET":            "blog",
		"S3_ACCESS_KEY":        "ak",
		"S3_SECRET_KEY":        "sk",
		"S3_PUBLIC_BASE_URL":   "https://cdn.example",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Admin == nil {
		t.Fatal("Admin is nil")
	}
	if !bytes.Equal(got.Admin.PublicKey, pub) {
		t.Error("public key does not match the PEM")
	}
	if strings.Join(got.Admin.Subjects, ",") != "github:1,github:2" {
		t.Errorf("subjects = %v", got.Admin.Subjects)
	}
	if got.Admin.S3.Endpoint != "acct.r2.cloudflarestorage.com" || !got.Admin.S3.Secure {
		t.Errorf("s3 = %+v, want the host with Secure", got.Admin.S3)
	}
}

func TestLoadAdminRejectsAPartialSet(t *testing.T) {
	t.Parallel()
	_, err := config.Load(env(map[string]string{
		"DATABASE_URL": "postgres://x/y",
		"S3_BUCKET":    "blog",
	}))
	if err == nil || !strings.Contains(err.Error(), "ADMIN_JWT_PUBLIC_KEY") {
		t.Fatalf("err = %v, want it to name a missing admin variable", err)
	}
}
