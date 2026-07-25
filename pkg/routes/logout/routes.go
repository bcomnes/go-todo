// Package logout owns browser and JSON token-revocation routes.
package logout

import (
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
	"github.com/danielgtaylor/huma/v2"
)

type routes struct {
	auth     *auth.Service
	sessions *httpx.Sessions
}

// Register adds the authenticated browser logout action to mux and the JSON operation to api.
func Register(mux *http.ServeMux, api huma.API, authService *auth.Service, sessions *httpx.Sessions) {
	routes := &routes{auth: authService, sessions: sessions}
	mux.HandleFunc("POST /logout", sessions.RequirePage(routes.postPage))
	huma.Register(api, huma.Operation{
		OperationID:   "logout",
		Method:        http.MethodPost,
		Path:          "/logout",
		Summary:       "Log out",
		Tags:          []string{"Authentication"},
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusUnauthorized},
	}, routes.postAPI)
}
