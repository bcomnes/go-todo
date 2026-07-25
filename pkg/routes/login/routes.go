package login

import (
	"net/http"
	"strings"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
	"github.com/bcomnes/go-todo/pkg/web"
	"github.com/bcomnes/go-todo/pkg/web/layout"
	"github.com/danielgtaylor/huma/v2"
)

type routes struct {
	auth     *auth.Service
	sessions *httpx.Sessions
	page     *web.Page
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register adds the login page and browser action to mux and the JSON operation to api.
func Register(mux *http.ServeMux, api huma.API, authService *auth.Service, sessions *httpx.Sessions) error {
	page, err := newPage()
	if err != nil {
		return err
	}
	routes := &routes{auth: authService, sessions: sessions, page: page}
	mux.HandleFunc("GET /login", routes.getPage)
	mux.HandleFunc("POST /login", routes.postPage)
	huma.Register(api, huma.Operation{
		OperationID:   "login",
		Method:        http.MethodPost,
		Path:          "/login",
		Summary:       "Log in",
		Tags:          []string{"Authentication"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusServiceUnavailable},
	}, routes.postAPI)
	return nil
}

func prepare(input *loginRequest) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
}

func (routes *routes) renderError(w http.ResponseWriter, r *http.Request, input loginRequest, status int, message string) {
	data := pageData{
		Data:  layout.Data{Title: "Log in"},
		Error: message,
		Email: input.Email,
	}
	httpx.RenderFormError(w, r, status, routes.page, "login-form", data)
}
