package landing

import (
	"net/http"

	"github.com/bcomnes/go-todo/pkg/httpx"
	"github.com/bcomnes/go-todo/pkg/web"
)

type routes struct {
	sessions *httpx.Sessions
	page     *web.Page
}

// Register adds the landing page and its fragment endpoint to mux.
func Register(mux *http.ServeMux, sessions *httpx.Sessions) error {
	page, err := newPage()
	if err != nil {
		return err
	}
	routes := &routes{sessions: sessions, page: page}
	mux.HandleFunc("GET /{$}", routes.get)
	return nil
}
