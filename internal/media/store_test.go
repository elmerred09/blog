package media

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	storeImage  = "rustfs/rustfs:1.0.1"
	storeBucket = "blog"
	accessKey   = "blog"
	secretKey   = "blogsecret"
)

// store is the shared RustFS, started in TestMain. Nil under -short.
var store *storeEnv

type storeEnv struct {
	endpoint string
	uploader *Uploader
	admin    *minio.Client
}

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		env, stop, err := startStore(ctx)
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "start rustfs: %v\n", err)
			os.Exit(1)
		}
		store = env
		code := m.Run()
		stop()
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func startStore(ctx context.Context) (*storeEnv, func(), error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Image:        storeImage,
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"RUSTFS_ACCESS_KEY":     accessKey,
			"RUSTFS_SECRET_KEY":     secretKey,
			"RUSTFS_ADDRESS":        ":9000",
			"RUSTFS_CONSOLE_ENABLE": "false",
		},
		Cmd:        []string{"/data"},
		WaitingFor: wait.ForHTTP("/health").WithPort("9000/tcp"),
		Started:    true,
	})
	stop := func() {
		if container != nil {
			_ = container.Terminate(context.Background())
		}
	}
	if err != nil {
		stop()
		return nil, nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		stop()
		return nil, nil, err
	}
	port, err := container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		stop()
		return nil, nil, err
	}
	endpoint := host + ":" + port.Port()

	uploader, err := New(Config{
		Endpoint:      endpoint,
		Region:        "us-east-1",
		Bucket:        storeBucket,
		AccessKey:     accessKey,
		SecretKey:     secretKey,
		PublicBaseURL: "http://" + endpoint + "/" + storeBucket,
	})
	if err != nil {
		stop()
		return nil, nil, err
	}
	if err := applyBucketConfig(ctx, uploader.client); err != nil {
		stop()
		return nil, nil, err
	}
	return &storeEnv{endpoint: endpoint, uploader: uploader, admin: uploader.client}, stop, nil
}

// applyBucketConfig creates the bucket and applies the same policy and CORS
// files compose's init container puts on the local store.
func applyBucketConfig(ctx context.Context, client *minio.Client) error {
	if err := client.MakeBucket(ctx, storeBucket, minio.MakeBucketOptions{Region: "us-east-1"}); err != nil {
		return fmt.Errorf("make bucket: %w", err)
	}

	policy, err := os.ReadFile("../../deploy/rustfs/bucket-policy.json")
	if err != nil {
		return err
	}
	// BUCKET in the ARN is a placeholder, filled in the way init.sh does.
	filled := strings.ReplaceAll(string(policy), "BUCKET", storeBucket)
	if err := client.SetBucketPolicy(ctx, storeBucket, filled); err != nil {
		return fmt.Errorf("set bucket policy: %w", err)
	}

	rules, err := corsRules("../../deploy/rustfs/cors.json")
	if err != nil {
		return err
	}
	if err := client.SetBucketCors(ctx, storeBucket, cors.NewConfig(rules)); err != nil {
		return fmt.Errorf("set bucket cors: %w", err)
	}
	return nil
}

func corsRules(name string) ([]cors.Rule, error) {
	raw, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	var file struct {
		CORSRules []struct {
			AllowedOrigins []string
			AllowedMethods []string
			AllowedHeaders []string
			MaxAgeSeconds  int
		}
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	rules := make([]cors.Rule, len(file.CORSRules))
	for i, r := range file.CORSRules {
		rules[i] = cors.Rule{
			AllowedOrigin: r.AllowedOrigins,
			AllowedMethod: r.AllowedMethods,
			AllowedHeader: r.AllowedHeaders,
			MaxAgeSeconds: r.MaxAgeSeconds,
		}
	}
	return rules, nil
}

// do sends req and returns the status, headers, and body. It closes the body.
func do(t *testing.T, req *http.Request) (int, http.Header, []byte) {
	t.Helper()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close body: %v", err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, resp.Header, body
}

func putSigned(t *testing.T, rawURL, contentType string, body []byte) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPut, rawURL, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", contentType)
	status, _, _ := do(t, req)
	return status
}

func TestStoreEnforcesTheSignature(t *testing.T) {
	if store == nil {
		t.Skip("no store under -short")
	}
	ctx := t.Context()

	const body = "png!"
	upload, err := store.uploader.PresignImageUpload(ctx, "image/png", int64(len(body)))
	if err != nil {
		t.Fatalf("presign: %v", err)
	}

	t.Run("a larger body", func(t *testing.T) {
		if status := putSigned(t, upload.URL, "image/png", []byte(body+"x")); status != http.StatusForbidden {
			t.Errorf("status = %d, want 403", status)
		}
	})

	t.Run("a smaller body", func(t *testing.T) {
		if status := putSigned(t, upload.URL, "image/png", []byte("png")); status != http.StatusForbidden {
			t.Errorf("status = %d, want 403", status)
		}
	})

	t.Run("a different content type", func(t *testing.T) {
		if status := putSigned(t, upload.URL, "text/html", []byte(body)); status != http.StatusForbidden {
			t.Errorf("status = %d, want 403", status)
		}
	})

	t.Run("the same signature on a different key", func(t *testing.T) {
		other := strings.Replace(upload.URL, path.Base(strings.Split(upload.URL, "?")[0]), "other.png", 1)
		if status := putSigned(t, other, "image/png", []byte(body)); status != http.StatusForbidden {
			t.Errorf("status = %d, want 403", status)
		}
	})

	t.Run("the signed upload", func(t *testing.T) {
		if status := putSigned(t, upload.URL, upload.Headers["Content-Type"], []byte(body)); status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, upload.PublicURL, nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		status, header, read := do(t, req)
		if status != http.StatusOK {
			t.Fatalf("anonymous GET status = %d, want 200", status)
		}
		if ct := header.Get("Content-Type"); ct != "image/png" {
			t.Errorf("content type = %q, want image/png", ct)
		}
		if string(read) != body {
			t.Errorf("body = %q, want %q", read, body)
		}
	})
}

func TestStoreRefusesAnonymousListingAndWrites(t *testing.T) {
	if store == nil {
		t.Skip("no store under -short")
	}
	ctx := t.Context()
	base := "http://" + store.endpoint + "/" + storeBucket

	if _, err := store.admin.PutObject(ctx, storeBucket, "notes/secret.txt", strings.NewReader("nope"), 4, minio.PutObjectOptions{ContentType: "text/plain"}); err != nil {
		t.Fatalf("put object outside images/: %v", err)
	}

	outside, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/notes/secret.txt", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if status, _, _ := do(t, outside); status != http.StatusForbidden {
		t.Errorf("GET outside images/ status = %d, want 403", status)
	}

	list, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"?list-type=2", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if status, _, _ := do(t, list); status != http.StatusForbidden {
		t.Errorf("list status = %d, want 403", status)
	}

	write, err := http.NewRequestWithContext(ctx, http.MethodPut, base+"/images/evil.png", strings.NewReader("nope"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if status, _, _ := do(t, write); status != http.StatusForbidden {
		t.Errorf("anonymous PUT status = %d, want 403", status)
	}
}

func TestStoreCORSAllowsTheDevOrigin(t *testing.T) {
	if store == nil {
		t.Skip("no store under -short")
	}
	target := "http://" + store.endpoint + "/" + storeBucket + "/images/x.png"

	preflight := func(origin string) http.Header {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodOptions, target, nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "PUT")
		req.Header.Set("Access-Control-Request-Headers", "content-type")
		_, header, _ := do(t, req)
		return header
	}

	if got := preflight("http://localhost:3000").Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("allow origin = %q, want http://localhost:3000", got)
	}
	if got := preflight("https://evil.example").Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("allow origin for an unknown site = %q, want none", got)
	}
}
