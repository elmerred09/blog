package httpapi

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/elmerred09/blog/internal/auth"
)

const specPath = "../../api/openapi.yaml"

type apiContract struct {
	doc    *openapi3.T
	router routers.Router
}

// loadContract parses and validates the spec once per test binary.
var loadContract = sync.OnceValues(func() (*apiContract, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", specPath, err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("invalid spec %s: %w", specPath, err)
	}
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("route spec %s: %w", specPath, err)
	}
	return &apiContract{doc: doc, router: router}, nil
})

func mustContract(t *testing.T) *apiContract {
	t.Helper()
	c, err := loadContract()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

var formatValidation = []openapi3.SchemaValidationOption{openapi3.EnableFormatValidation()}

// validateContract checks a response against the spec. A request the spec
// routes must get a status and body its operation documents. A request it
// doesn't route (an unknown path or a wrong method) must get a JSON Error if it
// failed. Anything else is a test-only route, which the spec doesn't cover.
func (c *apiContract) validateContract(req *http.Request, rec *httptest.ResponseRecorder) error {
	route, pathParams, err := c.router.FindRoute(req)
	if errors.Is(err, routers.ErrPathNotFound) || errors.Is(err, routers.ErrMethodNotAllowed) {
		if rec.Code < http.StatusBadRequest {
			return nil
		}
		var body any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			return fmt.Errorf("%s %s: unrouted %d response is not JSON: %w", req.Method, req.URL, rec.Code, err)
		}
		if err := c.doc.Components.Schemas["Error"].Value.VisitJSON(body, formatValidation...); err != nil {
			return fmt.Errorf("%s %s: unrouted %d response is not an Error: %w", req.Method, req.URL, rec.Code, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s %s: find route: %w", req.Method, req.URL, err)
	}

	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		},
		Status: rec.Code,
		Header: rec.Header(),
		Body:   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
		Options: &openapi3filter.Options{
			IncludeResponseStatus:   true,
			MultiError:              true,
			SchemaValidationOptions: formatValidation,
		},
	}
	if err := openapi3filter.ValidateResponse(req.Context(), input); err != nil {
		return fmt.Errorf("%s %s: %d response doesn't match the spec: %w", req.Method, req.URL, rec.Code, err)
	}
	return nil
}

// checkContract fails the test if the response doesn't match the spec. serve
// calls it for every request, so each test also checks the contract.
func checkContract(t *testing.T, req *http.Request, rec *httptest.ResponseRecorder) {
	t.Helper()
	if err := mustContract(t).validateContract(req, rec); err != nil {
		t.Error(err)
	}
}

var ginParam = regexp.MustCompile(`:(\w+)`)

func TestSpecMatchesRouter(t *testing.T) {
	t.Parallel()
	c := mustContract(t)
	log, _ := testLogger()

	// Admin routes exist only when configured, so the comparison uses an engine
	// that has them. The verifier is never called.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	verifier, err := auth.NewVerifier(pub, []string{"github:1"})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	var routed []string
	for _, r := range newEngine(Deps{Logger: log, Admin: &Admin{Verifier: verifier}}).Routes() {
		routed = append(routed, r.Method+" "+ginParam.ReplaceAllString(r.Path, "{$1}"))
	}
	var documented []string
	for path, item := range c.doc.Paths.Map() {
		for method := range item.Operations() {
			documented = append(documented, method+" "+path)
		}
	}
	slices.Sort(routed)
	slices.Sort(documented)

	if !slices.Equal(routed, documented) {
		t.Errorf("router and spec disagree:\n router %v\n   spec %v", routed, documented)
	}
}

func TestContractRejectsDrift(t *testing.T) {
	t.Parallel()
	c := mustContract(t)

	const validItem = `{"slug":"a","title":"A","summary":"s","published_at":"2026-10-01T12:00:00Z","tags":[]}`

	tests := []struct {
		name   string
		target string
		status int
		body   string
		ok     bool
	}{
		{"valid page", "/api/v1/posts", 200, `{"items":[` + validItem + `],"next_cursor":null}`, true},
		{"undocumented field", "/api/v1/posts", 200, `{"items":[],"next_cursor":null,"total":3}`, false},
		{"missing next_cursor", "/api/v1/posts", 200, `{"items":[]}`, false},
		{"tags null", "/api/v1/posts", 200, `{"items":[{"slug":"a","title":"A","summary":"s","published_at":"2026-10-01T12:00:00Z","tags":null}],"next_cursor":null}`, false},
		{"bad date-time", "/api/v1/posts", 200, `{"items":[{"slug":"a","title":"A","summary":"s","published_at":"yesterday","tags":[]}],"next_cursor":null}`, false},
		{"undocumented status", "/api/v1/posts", http.StatusTeapot, `{"error":{"code":"internal","message":"x"}}`, false},
		{"unknown error code", "/api/v1/posts/a", 404, `{"error":{"code":"gone","message":"x"}}`, false},
		{"unrouted 404 must be an Error", "/nope", 404, `not found`, false},
		{"unrouted 404 Error", "/nope", 404, `{"error":{"code":"not_found","message":"route not found"}}`, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.target, nil)
			rec := httptest.NewRecorder()
			rec.Header().Set("Content-Type", "application/json; charset=utf-8")
			rec.WriteHeader(tc.status)
			_, _ = rec.WriteString(tc.body)

			err := c.validateContract(req, rec)
			if tc.ok && err != nil {
				t.Errorf("valid response rejected: %v", err)
			}
			if !tc.ok && err == nil {
				t.Errorf("drifted response accepted")
			}
		})
	}
}
