// Package todos owns authenticated browser and JSON todo routes, their request
// parsing, and the server-rendered todo page and fragment.
package todos

import (
	"errors"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/httpx"
	todostore "github.com/bcomnes/go-todo/pkg/todos"
	"github.com/bcomnes/go-todo/pkg/web"
)

const (
	defaultListLimit = 20
	todoListFragment = "todo-list"
)

type routes struct {
	service  *todostore.Service
	sessions *httpx.Sessions
	page     *web.Page
}

// Register adds authenticated browser and JSON todo routes to mux.
func Register(mux *http.ServeMux, service *todostore.Service, sessions *httpx.Sessions) error {
	if mux == nil {
		return errors.New("todo routes require a mux")
	}
	if service == nil {
		return errors.New("todo routes require a service")
	}
	if sessions == nil {
		return errors.New("todo routes require sessions")
	}
	page, err := newPage()
	if err != nil {
		return err
	}
	routes := &routes{service: service, sessions: sessions, page: page}

	mux.HandleFunc("GET /todos", sessions.RequirePage(routes.getPage))
	mux.HandleFunc("POST /todos", routes.pageMutation(routes.createPage))
	mux.HandleFunc("POST /todos/{id}", routes.pageMutation(routes.updatePage))
	mux.HandleFunc("POST /todos/{id}/toggle", routes.pageMutation(routes.togglePage))
	mux.HandleFunc("POST /todos/{id}/delete", routes.pageMutation(routes.deletePage))

	mux.HandleFunc("GET /api/todos", sessions.RequireAPI(routes.listAPI))
	mux.HandleFunc("POST /api/todos", sessions.RequireAPI(routes.createAPI))
	mux.HandleFunc("GET /api/todos/{id}", sessions.RequireAPI(routes.getAPI))
	mux.HandleFunc("PATCH /api/todos/{id}", sessions.RequireAPI(routes.updateAPI))
	mux.HandleFunc("DELETE /api/todos/{id}", sessions.RequireAPI(routes.deleteAPI))
	return nil
}

func (routes *routes) pageMutation(next http.HandlerFunc) http.HandlerFunc {
	return routes.sessions.RequirePage(func(w http.ResponseWriter, r *http.Request) {
		if !routes.sessions.HasSameOrigin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}
