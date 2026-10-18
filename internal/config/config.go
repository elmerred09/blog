// Package config loads settings from the environment.
package config

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/elmerred09/blog/internal/auth"
	"github.com/elmerred09/blog/internal/media"
)

// DefaultHTTPAddr is used when HTTP_ADDR is unset.
const DefaultHTTPAddr = ":8080"

// adminEnv is every variable the admin routes need. All of them or none:
// a partial set is a mistake, and an empty set means the public API runs
// without the editor.
var adminEnv = []string{
	"ADMIN_JWT_PUBLIC_KEY",
	"ADMIN_SUBJECTS",
	"S3_ENDPOINT",
	"S3_REGION",
	"S3_BUCKET",
	"S3_ACCESS_KEY",
	"S3_SECRET_KEY",
	"S3_PUBLIC_BASE_URL",
}

type Config struct {
	DatabaseURL string
	HTTPAddr    string
	// Admin is nil when no admin variable is set. The public API still starts.
	Admin *Admin
}

// Admin is the editor's token check and the bucket it signs uploads for.
type Admin struct {
	PublicKey ed25519.PublicKey
	Subjects  []string
	S3        media.Config
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

	admin, err := loadAdmin(getenv)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.Admin = admin

	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

func loadAdmin(getenv func(string) string) (*Admin, error) {
	var set, missing []string
	for _, name := range adminEnv {
		if getenv(name) == "" {
			missing = append(missing, name)
		} else {
			set = append(set, name)
		}
	}
	if len(set) == 0 {
		return nil, nil
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("admin config is partial (%s are set; missing %s)",
			strings.Join(set, ", "), strings.Join(missing, ", "))
	}

	key, err := auth.ParsePublicKey([]byte(unescape(getenv("ADMIN_JWT_PUBLIC_KEY"))))
	if err != nil {
		return nil, fmt.Errorf("ADMIN_JWT_PUBLIC_KEY: %w", err)
	}

	var subjects []string
	for _, s := range strings.Split(getenv("ADMIN_SUBJECTS"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, errors.New("ADMIN_SUBJECTS contains an empty subject")
		}
		subjects = append(subjects, s)
	}

	endpoint, secure, err := s3Endpoint(getenv("S3_ENDPOINT"))
	if err != nil {
		return nil, fmt.Errorf("S3_ENDPOINT: %w", err)
	}

	return &Admin{
		PublicKey: key,
		Subjects:  subjects,
		S3: media.Config{
			Endpoint:      endpoint,
			Region:        getenv("S3_REGION"),
			Bucket:        getenv("S3_BUCKET"),
			AccessKey:     getenv("S3_ACCESS_KEY"),
			SecretKey:     getenv("S3_SECRET_KEY"),
			PublicBaseURL: getenv("S3_PUBLIC_BASE_URL"),
			Secure:        secure,
		},
	}, nil
}

// unescape turns the \n sequences of a PEM value written on one env line
// into real newlines.
func unescape(s string) string {
	return strings.NewReplacer(`\n`, "\n").Replace(s)
}

// s3Endpoint accepts host:port or an http(s) URL. https turns Secure on.
func s3Endpoint(raw string) (endpoint string, secure bool, err error) {
	if !strings.Contains(raw, "://") {
		return raw, false, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false, err
	}
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", false, fmt.Errorf("%q needs an http or https scheme", raw)
	}
	return u.Host, u.Scheme == "https", nil
}
