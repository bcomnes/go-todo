package account

import (
	"net/http"

	"github.com/bcomnes/go-todo/pkg/httpx"
	"github.com/bcomnes/go-todo/pkg/web"
)

type routes struct {
	sessions *httpx.Sessions
	page     *web.Page
}

// Register adds the authenticated account page and JSON endpoint to mux.
func Register(mux *http.ServeMux, sessions *httpx.Sessions) error {
	page, err := newPage()
	if err != nil {
		return err
	}
	routes := &routes{sessions: sessions, page: page}
	mux.HandleFunc("GET /account", sessions.RequirePage(routes.getPage))
	mux.HandleFunc("GET /api/account", sessions.RequireAPI(routes.getAPI))
	return nil
}
