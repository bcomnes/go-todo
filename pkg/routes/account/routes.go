package account

import (
	"net/http"

	"github.com/bcomnes/go-todo/pkg/httpx"
	"github.com/bcomnes/go-todo/pkg/web"
	"github.com/danielgtaylor/huma/v2"
)

type routes struct {
	sessions *httpx.Sessions
	page     *web.Page
}

// Register adds the authenticated account page to mux and the JSON operation to api.
func Register(mux *http.ServeMux, api huma.API, sessions *httpx.Sessions) error {
	page, err := newPage()
	if err != nil {
		return err
	}
	routes := &routes{sessions: sessions, page: page}
	mux.HandleFunc("GET /account", sessions.RequirePage(routes.getPage))
	huma.Register(api, huma.Operation{
		OperationID:   "get-account",
		Method:        http.MethodGet,
		Path:          "/account",
		Summary:       "Get the current account",
		Tags:          []string{"Account"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusUnauthorized, http.StatusServiceUnavailable},
	}, routes.getAPI)
	return nil
}
