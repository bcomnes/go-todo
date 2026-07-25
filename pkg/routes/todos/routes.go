// Package todos owns authenticated browser and JSON todo routes, their request
// parsing, and the server-rendered todo page and fragment.
package todos

import (
	"errors"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/httpx"
	todostore "github.com/bcomnes/go-todo/pkg/todos"
	"github.com/bcomnes/go-todo/pkg/web"
	"github.com/danielgtaylor/huma/v2"
)

const defaultListLimit = 20

type routes struct {
	service    *todostore.Service
	sessions   *httpx.Sessions
	indexPage  *web.Page
	detailPage *web.Page
	editPage   *web.Page
}

// Register adds authenticated browser and JSON todo routes to mux.
func Register(mux *http.ServeMux, api huma.API, service *todostore.Service, sessions *httpx.Sessions) error {
	if mux == nil {
		return errors.New("todo routes require a mux")
	}
	if service == nil {
		return errors.New("todo routes require a service")
	}
	if sessions == nil {
		return errors.New("todo routes require sessions")
	}
	indexPage, detailPage, editPage, err := newPages()
	if err != nil {
		return err
	}
	routes := &routes{
		service:    service,
		sessions:   sessions,
		indexPage:  indexPage,
		detailPage: detailPage,
		editPage:   editPage,
	}

	mux.HandleFunc("GET /todos", sessions.RequirePage(routes.getPage))
	mux.HandleFunc(TodoDetailPagePattern, sessions.RequirePage(routes.GetDetailPage))
	mux.HandleFunc(TodoEditFormPattern, sessions.RequirePage(routes.GetEditForm))
	mux.HandleFunc("POST /todos", routes.pageMutation(routes.createPage))
	mux.HandleFunc("POST /todos/{id}", routes.pageMutation(routes.updatePage))
	mux.HandleFunc("POST /todos/{id}/toggle", routes.pageMutation(routes.togglePage))
	mux.HandleFunc("POST /todos/{id}/delete", routes.pageMutation(routes.deletePage))

	huma.Register(api, huma.Operation{
		OperationID:   "list-todos",
		Method:        http.MethodGet,
		Path:          "/todos",
		Summary:       "List todos",
		Tags:          []string{"Todos"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusUnauthorized, http.StatusBadRequest, http.StatusServiceUnavailable},
	}, routes.listAPI)
	huma.Register(api, huma.Operation{
		OperationID:   "create-todo",
		Method:        http.MethodPost,
		Path:          "/todos",
		Summary:       "Create a todo",
		Tags:          []string{"Todos"},
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnauthorized, http.StatusBadRequest, http.StatusServiceUnavailable},
	}, routes.createAPI)
	huma.Register(api, huma.Operation{
		OperationID:   "get-todo",
		Method:        http.MethodGet,
		Path:          "/todos/{id}",
		Summary:       "Get a todo",
		Tags:          []string{"Todos"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusServiceUnavailable},
	}, routes.getAPI)
	huma.Register(api, huma.Operation{
		OperationID:   "update-todo",
		Method:        http.MethodPatch,
		Path:          "/todos/{id}",
		Summary:       "Update a todo",
		Tags:          []string{"Todos"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusUnauthorized, http.StatusBadRequest, http.StatusNotFound, http.StatusServiceUnavailable},
	}, routes.updateAPI)
	huma.Register(api, huma.Operation{
		OperationID:   "delete-todo",
		Method:        http.MethodDelete,
		Path:          "/todos/{id}",
		Summary:       "Delete a todo",
		Tags:          []string{"Todos"},
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusServiceUnavailable},
	}, routes.deleteAPI)
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
