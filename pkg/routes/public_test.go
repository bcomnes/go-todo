package routes_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bcomnes/go-todo/pkg/httpapi"
)

func TestHealth(t *testing.T) {
	api, err := httpapi.New(&sql.DB{}, time.Hour)
	if err != nil {
		t.Fatalf("initialize API: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status body = %q, want ok", body["status"])
	}
}

func TestPublicPagesAndAssets(t *testing.T) {
	api, err := httpapi.New(&sql.DB{}, time.Hour)
	if err != nil {
		t.Fatalf("initialize API: %v", err)
	}
	handler := api.Handler()

	tests := []struct {
		path        string
		contentType string
		contains    string
	}{
		{path: "/", contentType: "text/html", contains: "Keep your todos focused"},
		{path: "/login", contentType: "text/html", contains: `id="login-form"`},
		{path: "/register", contentType: "text/html", contains: `id="register-form"`},
		{path: "/assets/global.css", contentType: "text/css", contains: "mine"},
		{path: "/assets/global.js", contentType: "text/javascript", contains: "htmx"},
		{path: "/assets/pages/todos/index.js", contentType: "text/javascript", contains: "chunks/"},
		{path: "/assets/pages/todos/detail.js", contentType: "text/javascript", contains: "chunks/"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, test.contentType) {
				t.Fatalf("content type = %q, want prefix %q", contentType, test.contentType)
			}
			if !strings.Contains(response.Body.String(), test.contains) {
				t.Fatalf("response does not contain %q", test.contains)
			}
		})
	}
}

func TestHumaServesCompleteOpenAPIContract(t *testing.T) {
	api, err := httpapi.New(&sql.DB{}, time.Hour)
	if err != nil {
		t.Fatalf("initialize API: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("OpenAPI status = %d, body = %s", response.Code, response.Body.String())
	}

	var document struct {
		OpenAPI string `json:"openapi"`
		Paths   map[string]map[string]struct {
			OperationID string                `json:"operationId"`
			Security    []map[string][]string `json:"security"`
		} `json:"paths"`
		Components struct {
			SecuritySchemes map[string]struct {
				Type   string `json:"type"`
				Scheme string `json:"scheme"`
			} `json:"securitySchemes"`
		} `json:"components"`
	}
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		t.Fatalf("decode OpenAPI document: %v", err)
	}
	if document.OpenAPI != "3.1.0" {
		t.Fatalf("OpenAPI version = %q, want 3.1.0", document.OpenAPI)
	}
	for _, path := range []string{"/login", "/register", "/logout", "/account", "/todos", "/todos/{id}"} {
		if document.Paths[path] == nil {
			t.Errorf("OpenAPI document is missing path %q", path)
		}
	}
	if got := document.Paths["/todos/{id}"]["patch"].OperationID; got != "update-todo" {
		t.Errorf("PATCH todo operation ID = %q, want update-todo", got)
	}
	if len(document.Paths["/todos"]["get"].Security) == 0 {
		t.Error("list todos operation does not declare bearer security")
	}
	bearer := document.Components.SecuritySchemes["bearer"]
	if bearer.Type != "http" || bearer.Scheme != "bearer" {
		t.Errorf("bearer security scheme = %#v", bearer)
	}
}

func TestHumaDocsAreMountedUnderAPIPrefix(t *testing.T) {
	api, err := httpapi.New(&sql.DB{}, time.Hour)
	if err != nil {
		t.Fatalf("initialize API: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("docs status = %d, body = %s", response.Code, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("docs content type = %q, want HTML", contentType)
	}
}

func TestHTMXFormValidationRendersColocatedFragments(t *testing.T) {
	api, err := httpapi.New(&sql.DB{}, time.Hour)
	if err != nil {
		t.Fatalf("initialize API: %v", err)
	}
	handler := api.Handler()

	tests := []struct {
		path       string
		body       string
		wantStatus int
		contains   string
	}{
		{path: "/login", body: "email=&password=", wantStatus: http.StatusUnauthorized, contains: `id="login-form"`},
		{path: "/register", body: "username=x&email=invalid&password=short", wantStatus: http.StatusBadRequest, contains: `id="register-form"`},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.Header.Set("Origin", "https://example.com")
			request.Header.Set("HX-Request", "true")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), test.contains) || !strings.Contains(response.Body.String(), `role="alert"`) {
				t.Fatalf("validation fragment is incomplete: %s", response.Body.String())
			}
			if strings.Contains(response.Body.String(), "<!doctype html>") {
				t.Fatal("validation fragment contains the shared layout")
			}
		})
	}
}

func TestBrowserActionsRejectCrossOriginRequests(t *testing.T) {
	api, err := httpapi.New(&sql.DB{}, time.Hour)
	if err != nil {
		t.Fatalf("initialize API: %v", err)
	}
	for _, origin := range []string{"https://attacker.example", "http://example.com"} {
		t.Run(origin, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("email=user@example.test&password=password"))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.Header.Set("Origin", origin)
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, request)

			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
			}
		})
	}
}

func TestJSONBodyRoutesRequireJSON(t *testing.T) {
	api, err := httpapi.New(&sql.DB{}, time.Hour)
	if err != nil {
		t.Fatalf("initialize API: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"email":"user@example.test","password":"password"}`))
	request.Header.Set("Content-Type", "text/plain")
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnsupportedMediaType)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/problem+json") {
		t.Fatalf("content type = %q, want Problem Details JSON", contentType)
	}
	if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("cache control = %q, want no-store", cacheControl)
	}
}

func TestLandingFragment(t *testing.T) {
	api, err := httpapi.New(&sql.DB{}, time.Hour)
	if err != nil {
		t.Fatalf("initialize API: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/?fragment=health-status", nil)
	request.Header.Set("HX-Request", "true")
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "<!doctype html>") {
		t.Fatal("fragment response contains the shared document layout")
	}
	if !strings.Contains(response.Body.String(), `id="health-status"`) {
		t.Fatal("fragment response does not contain the health status")
	}
}
