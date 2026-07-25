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
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// New builds the complete browser, asset, health, and JSON route tree.
func New(
	authService *auth.Service,
	sessions *httpx.Sessions,
	todoService *todostore.Service,
) (http.Handler, error) {
	mux := http.NewServeMux()
	config := huma.DefaultConfig("go-todo API", "1.0.0")
	config.RejectUnknownQueryParameters = true
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "opaque",
		},
	}
	api := humago.NewWithPrefix(mux, "/api", config)
	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		ctx.SetHeader("Cache-Control", "no-store")
		next(ctx)
	})
	protectedAPI := huma.NewGroup(api)
	protectedAPI.UseMiddleware(sessions.RequireHuma(api))
	protectedAPI.UseSimpleModifier(func(operation *huma.Operation) {
		operation.Security = []map[string][]string{{"bearer": {}}}
	})

	assets := http.StripPrefix("/assets/", http.FileServer(http.FS(web.Assets())))
	mux.Handle("GET /assets/", cacheAssets(assets))
	health.Register(mux)
	if err := landing.Register(mux, sessions); err != nil {
		return nil, err
	}
	if err := login.Register(mux, api, authService, sessions); err != nil {
		return nil, err
	}
	if err := register.Register(mux, api, authService, sessions); err != nil {
		return nil, err
	}
	if err := account.Register(mux, protectedAPI, sessions); err != nil {
		return nil, err
	}
	logout.Register(mux, protectedAPI, authService, sessions)
	if err := todoroutes.Register(mux, protectedAPI, todoService, sessions); err != nil {
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
