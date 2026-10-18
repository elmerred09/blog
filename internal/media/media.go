// Package media signs browser uploads of images straight to S3-compatible
// storage. The file bytes never pass through this process: PresignImageUpload
// returns a URL the browser PUTs to, and the signature fixes the key, the
// content type, and the exact size.
package media

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	// MaxBytes is the largest upload that will be signed.
	MaxBytes = 10 << 20
	// Expiry is how long a signed URL stays valid.
	Expiry = 5 * time.Minute

	keyPrefix = "images/"
)

// extensions are the only content types that may be uploaded. SVG is absent
// because it can carry script. The extension is chosen here, not by the caller.
var extensions = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/webp": "webp",
	"image/gif":  "gif",
}

// Config is how to reach the bucket and how to build a public URL for an
// object. Endpoint is host:port with no scheme. PublicBaseURL is the base
// visitors use, such as http://127.0.0.1:9000/blog locally or the bucket's
// custom domain on R2, with no trailing slash required.
type Config struct {
	Endpoint      string
	Region        string
	Bucket        string
	AccessKey     string
	SecretKey     string
	PublicBaseURL string
	Secure        bool
}

// Problem is one input that failed validation. Message is safe to show.
type Problem struct {
	Field   string
	Message string
}

// ValidationError is an upload request that will not be signed. Problems
// lists every failure.
type ValidationError struct {
	Problems []Problem
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		parts[i] = p.Field + ": " + p.Message
	}
	return "invalid upload: " + strings.Join(parts, "; ")
}

// Upload is what the browser needs. URL is the signed PUT. Headers must be
// sent exactly: the signature covers Content-Type and Content-Length, so a
// different type or size is rejected by the store. PublicURL is the anonymous
// address to store in the post.
type Upload struct {
	URL       string
	Headers   map[string]string
	PublicURL string
}

type Uploader struct {
	client *minio.Client
	bucket string
	public string
}

func New(cfg Config) (*Uploader, error) {
	if cfg.Endpoint == "" || cfg.Region == "" || cfg.Bucket == "" ||
		cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.PublicBaseURL == "" {
		return nil, errors.New("media: endpoint, region, bucket, credentials, and public base URL are required")
	}
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       cfg.Secure,
		Region:       cfg.Region,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("media client: %w", err)
	}
	return &Uploader{
		client: client,
		bucket: cfg.Bucket,
		public: strings.TrimRight(cfg.PublicBaseURL, "/"),
	}, nil
}

// PresignImageUpload signs a PUT for one image. The object key is generated
// here, under images/, so the caller cannot choose it.
func (u *Uploader) PresignImageUpload(ctx context.Context, contentType string, size int64) (Upload, error) {
	var problems []Problem
	ext, ok := extensions[contentType]
	if !ok {
		problems = append(problems, Problem{
			Field:   "content_type",
			Message: "content type must be image/png, image/jpeg, image/webp, or image/gif",
		})
	}
	if size < 1 || size > MaxBytes {
		problems = append(problems, Problem{
			Field:   "size",
			Message: fmt.Sprintf("size must be from 1 byte to %d", MaxBytes),
		})
	}
	if len(problems) > 0 {
		return Upload{}, &ValidationError{Problems: problems}
	}

	id, err := uuid.NewV7()
	if err != nil {
		return Upload{}, fmt.Errorf("new image id: %w", err)
	}
	key := keyPrefix + id.String() + "." + ext

	length := strconv.FormatInt(size, 10)
	signed, err := u.client.PresignHeader(ctx, http.MethodPut, u.bucket, key, Expiry, nil, http.Header{
		"Content-Type":   []string{contentType},
		"Content-Length": []string{length},
	})
	if err != nil {
		return Upload{}, fmt.Errorf("presign %s: %w", key, err)
	}

	return Upload{
		URL: signed.String(),
		Headers: map[string]string{
			"Content-Type":   contentType,
			"Content-Length": length,
		},
		PublicURL: u.public + "/" + key,
	}, nil
}
