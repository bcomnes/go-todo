// Package httpapi is the public composition facade for go-todo's HTTP server.
// Feature handlers, templates, and route registration live under pkg/routes;
// this package constructs their shared authentication and session dependencies.
package httpapi

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
	"github.com/bcomnes/go-todo/pkg/routes"
	"github.com/bcomnes/go-todo/pkg/todos"
)

// Options controls security-sensitive API behavior. Zero values select secure
// cookies and a small bounded credential-hashing concurrency limit.
type Options struct {
	AllowInsecureCookies bool
	HashConcurrency      int
}

// API owns the fully composed HTTP handler.
type API struct {
	handler http.Handler
}

// New creates an API using production-safe defaults.
func New(db *sql.DB, tokenTTL time.Duration) (*API, error) {
	return NewWithOptions(db, tokenTTL, Options{})
}

// NewWithOptions creates the shared services and explicitly registers every
// feature route package.
func NewWithOptions(db *sql.DB, tokenTTL time.Duration, options Options) (*API, error) {
	authService, err := auth.New(db, tokenTTL, auth.Options{HashConcurrency: options.HashConcurrency})
	if err != nil {
		return nil, err
	}
	sessions, err := httpx.NewSessions(authService, !options.AllowInsecureCookies)
	if err != nil {
		return nil, err
	}
	todoService, err := todos.New(db)
	if err != nil {
		return nil, err
	}
	handler, err := routes.New(authService, sessions, todoService)
	if err != nil {
		return nil, err
	}
	return &API{handler: handler}, nil
}

// Handler returns the complete browser, asset, health, and JSON route tree.
func (api *API) Handler() http.Handler {
	return api.handler
}
