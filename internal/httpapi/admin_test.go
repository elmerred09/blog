package httpapi

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elmerred09/blog/internal/admin"
	"github.com/elmerred09/blog/internal/auth"
	"github.com/elmerred09/blog/internal/media"
	"github.com/elmerred09/blog/internal/platform/db/dbtest"
	"github.com/elmerred09/blog/internal/post"
	"github.com/elmerred09/blog/internal/render"
	"github.com/elmerred09/blog/internal/tags"
)

func signToken(t *testing.T, key ed25519.PrivateKey, sub string) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"iss": auth.Issuer,
		"aud": auth.Audience,
		"sub": sub,
		"exp": time.Now().Add(time.Minute).Unix(),
	}).SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

type adminAPI struct {
	h         http.Handler
	pool      *pgxpool.Pool
	token     string
	forbidden string
}

func newAdminAPI(t *testing.T) adminAPI {
	t.Helper()
	pool, _ := env.NewDB(t)
	log, _ := testLogger()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	verifier, err := auth.NewVerifier(pub, []string{testSubject})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	uploader, err := media.New(media.Config{
		Endpoint:      "127.0.0.1:9",
		Region:        "us-east-1",
		Bucket:        "blog",
		AccessKey:     "blog",
		SecretKey:     "blogsecret",
		PublicBaseURL: "http://127.0.0.1:9000/blog",
	})
	if err != nil {
		t.Fatalf("media.New: %v", err)
	}
	h := New(Deps{
		Logger: log,
		DB:     pool,
		Posts:  post.NewService(pool),
		Tags:   tags.NewStore(pool),
		Admin: &Admin{
			Verifier: verifier,
			Posts:    admin.NewService(pool, render.New()),
			Media:    uploader,
		},
	})
	return adminAPI{
		h:         h,
		pool:      pool,
		token:     signToken(t, priv, testSubject),
		forbidden: signToken(t, priv, "github:999"),
	}
}

func adminRequest(t *testing.T, method, target, token string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("marshal: %v", err)
		}
	}
	req := httptest.NewRequestWithContext(t.Context(), method, target, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func postBody(slug, title string) map[string]any {
	return map[string]any{
		"slug":    slug,
		"title":   title,
		"summary": "Summary of " + slug,
		"body_md": "# " + slug,
		"tags":    []string{"Go"},
	}
}

func TestAdminRequiresAuth(t *testing.T) {
	t.Parallel()
	api := newAdminAPI(t)

	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/admin/posts"},
		{http.MethodPost, "/api/v1/admin/posts"},
		{http.MethodGet, "/api/v1/admin/posts/" + uuid.NewString()},
		{http.MethodPut, "/api/v1/admin/posts/" + uuid.NewString()},
		{http.MethodPost, "/api/v1/admin/preview"},
		{http.MethodPost, "/api/v1/admin/uploads"},
		{http.MethodGet, "/api/v1/admin/tags"},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			missing := serve(t, api.h, adminRequest(t, rt.method, rt.path, "", nil))
			wantError(t, missing, http.StatusUnauthorized, codeUnauthorized)
			if missing.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want Bearer", missing.Header().Get("WWW-Authenticate"))
			}

			denied := serve(t, api.h, adminRequest(t, rt.method, rt.path, api.forbidden, nil))
			wantError(t, denied, http.StatusForbidden, codeForbidden)
		})
	}
}

func TestAdminPostRoundTrip(t *testing.T) {
	t.Parallel()
	api := newAdminAPI(t)

	created := serve(t, api.h, adminRequest(t, http.MethodPost, "/api/v1/admin/posts", api.token, postBody("hello", "Hello")))
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body %s", created.Code, created.Body)
	}
	var post struct {
		ID        string `json:"id"`
		UpdatedAt string `json:"updated_at"`
		State     string `json:"state"`
		BodyMD    string `json:"body_md"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &post); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if post.State != "draft" || post.BodyMD != "# hello" || post.UpdatedAt == "" {
		t.Fatalf("created = %+v", post)
	}

	// Send the updated_at string exactly as the JSON had it. Parsing it into a
	// time and formatting it again can drop a microsecond and cause a false 409.
	body := postBody("hello", "Hello again")
	body["updated_at"] = post.UpdatedAt
	saved := serve(t, api.h, adminRequest(t, http.MethodPut, "/api/v1/admin/posts/"+post.ID, api.token, body))
	if saved.Code != http.StatusOK {
		t.Fatalf("update status = %d, body %s", saved.Code, saved.Body)
	}

	var again struct {
		Title     string `json:"title"`
		UpdatedAt string `json:"updated_at"`
	}
	if err := json.Unmarshal(saved.Body.Bytes(), &again); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if again.Title != "Hello again" || again.UpdatedAt == post.UpdatedAt {
		t.Fatalf("saved = %+v, previous token %s", again, post.UpdatedAt)
	}

	stale := postBody("hello", "Too late")
	stale["updated_at"] = post.UpdatedAt
	conflict := serve(t, api.h, adminRequest(t, http.MethodPut, "/api/v1/admin/posts/"+post.ID, api.token, stale))
	wantError(t, conflict, http.StatusConflict, codeConflict)

	taken := postBody("hello", "Other")
	if rec := serve(t, api.h, adminRequest(t, http.MethodPost, "/api/v1/admin/posts", api.token, taken)); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, body %s", rec.Code, rec.Body)
	} else {
		wantError(t, rec, http.StatusConflict, codeSlugTaken)
	}

	got := serve(t, api.h, adminRequest(t, http.MethodGet, "/api/v1/admin/posts/"+post.ID, api.token, nil))
	if got.Code != http.StatusOK {
		t.Fatalf("get status = %d, body %s", got.Code, got.Body)
	}

	missing := serve(t, api.h, adminRequest(t, http.MethodGet, "/api/v1/admin/posts/"+uuid.NewString(), api.token, nil))
	wantError(t, missing, http.StatusNotFound, codeNotFound)
}

func TestAdminValidationAndPreview(t *testing.T) {
	t.Parallel()
	api := newAdminAPI(t)

	bad := postBody("Hello", "Hello")
	bad["title"] = ""
	rec := serve(t, api.h, adminRequest(t, http.MethodPost, "/api/v1/admin/posts", api.token, bad))
	wantError(t, rec, http.StatusBadRequest, codeInvalidRequest)
	var body struct {
		Error struct {
			Problems []struct{ Field string } `json:"problems"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Error.Problems) == 0 || body.Error.Problems[0].Field != "title" {
		t.Errorf("problems = %+v, want the title first", body.Error.Problems)
	}

	preview := serve(t, api.h, adminRequest(t, http.MethodPost, "/api/v1/admin/preview", api.token, map[string]any{"body_md": "# Hi"}))
	if preview.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body %s", preview.Code, preview.Body)
	}
	var rendered struct {
		BodyHTML string `json:"body_html"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &rendered); err != nil || !strings.Contains(rendered.BodyHTML, "<h1") {
		t.Fatalf("preview = %s", preview.Body)
	}

	upload := serve(t, api.h, adminRequest(t, http.MethodPost, "/api/v1/admin/uploads", api.token, map[string]any{
		"content_type": "image/png",
		"size":         4,
	}))
	if upload.Code != http.StatusOK || !strings.Contains(upload.Body.String(), "content-length") {
		t.Fatalf("upload = %d %s", upload.Code, upload.Body)
	}

	svg := serve(t, api.h, adminRequest(t, http.MethodPost, "/api/v1/admin/uploads", api.token, map[string]any{
		"content_type": "image/svg+xml",
		"size":         4,
	}))
	wantError(t, svg, http.StatusBadRequest, codeInvalidRequest)
}

func TestAdminBodyLimit(t *testing.T) {
	t.Parallel()
	api := newAdminAPI(t)

	under := postBody("sized", "Sized")
	under["body_md"] = strings.Repeat("a", admin.MaxBodyBytes)
	ok := serve(t, api.h, adminRequest(t, http.MethodPost, "/api/v1/admin/posts", api.token, under))
	if ok.Code != http.StatusCreated {
		t.Fatalf("body at the markdown limit: status = %d, body %s", ok.Code, ok.Body)
	}

	over := postBody("over", "Over")
	over["body_md"] = strings.Repeat("b", admin.MaxBodyBytes+1)
	invalid := serve(t, api.h, adminRequest(t, http.MethodPost, "/api/v1/admin/posts", api.token, over))
	wantError(t, invalid, http.StatusBadRequest, codeInvalidRequest)

	// A body of x's fails JSON parsing before the size cap is reached. A long
	// string forces the decoder to read past the cap.
	raw := append([]byte(`"`), bytes.Repeat([]byte("a"), maxJSONBody)...)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/admin/posts", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+api.token)
	req.Header.Set("Content-Type", "application/json")
	tooBig := serve(t, api.h, req)
	wantError(t, tooBig, http.StatusRequestEntityTooLarge, codeTooLarge)
}

func TestListTags(t *testing.T) {
	t.Parallel()
	api := newAdminAPI(t)
	dbtest.InsertPost(t, api.pool, dbtest.WithTags("go", "postgres"))
	dbtest.InsertPost(t, api.pool, dbtest.Draft(), dbtest.WithTags("drafts-only"))
	dbtest.InsertTag(t, api.pool, "unused")

	tests := []struct {
		name   string
		target string
		token  string
		want   string
	}{
		{"public hides draft-only and unused tags", "/api/v1/tags", "", "go,postgres"},
		{"admin lists every tag", "/api/v1/admin/tags", api.token, "drafts-only,go,postgres,unused"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := serve(t, api.h, adminRequest(t, http.MethodGet, tc.target, tc.token, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
			}
			var body struct {
				Tags []struct{ Slug string } `json:"tags"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			slugs := make([]string, len(body.Tags))
			for i, tag := range body.Tags {
				slugs[i] = tag.Slug
			}
			if got := strings.Join(slugs, ","); got != tc.want {
				t.Errorf("tags = %s, want %s", got, tc.want)
			}
		})
	}
}
