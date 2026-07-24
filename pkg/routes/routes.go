// Package routes composes go-todo's feature-oriented HTTP route packages.
package routes

import (
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
	"github.com/bcomnes/go-todo/pkg/routes/account"
	"github.com/bcomnes/go-todo/pkg/routes/health"
	"github.com/bcomnes/go-todo/pkg/routes/landing"
	"github.com/bcomnes/go-todo/pkg/routes/login"
	"github.com/bcomnes/go-todo/pkg/routes/logout"
	"github.com/bcomnes/go-todo/pkg/routes/register"
	todoroutes "github.com/bcomnes/go-todo/pkg/routes/todos"
	todostore "github.com/bcomnes/go-todo/pkg/todos"
	"github.com/bcomnes/go-todo/pkg/web"
)

// New builds the complete browser, asset, health, and JSON route tree.
func New(
	authService *auth.Service,
	sessions *httpx.Sessions,
	todoService *todostore.Service,
) (http.Handler, error) {
	mux := http.NewServeMux()
	assets := http.StripPrefix("/assets/", http.FileServer(http.FS(web.Assets())))
	mux.Handle("GET /assets/global.css", cacheAssets(assets))
	mux.Handle("GET /assets/global.js", cacheAssets(assets))
	health.Register(mux)
	if err := landing.Register(mux, sessions); err != nil {
		return nil, err
	}
	if err := login.Register(mux, authService, sessions); err != nil {
		return nil, err
	}
	if err := register.Register(mux, authService, sessions); err != nil {
		return nil, err
	}
	if err := account.Register(mux, sessions); err != nil {
		return nil, err
	}
	logout.Register(mux, authService, sessions)
	if err := todoroutes.Register(mux, todoService, sessions); err != nil {
		return nil, err
	}
	return mux, nil
}

func cacheAssets(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
		next.ServeHTTP(w, r)
	})
}
