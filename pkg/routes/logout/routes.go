// Package logout owns browser and JSON token-revocation routes.
package logout

import (
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
)

type routes struct {
	auth     *auth.Service
	sessions *httpx.Sessions
}

// Register adds authenticated browser and JSON logout actions to mux.
func Register(mux *http.ServeMux, authService *auth.Service, sessions *httpx.Sessions) {
	routes := &routes{auth: authService, sessions: sessions}
	mux.HandleFunc("POST /logout", sessions.RequirePage(routes.postPage))
	mux.HandleFunc("POST /api/logout", sessions.RequireAPI(routes.postAPI))
}
