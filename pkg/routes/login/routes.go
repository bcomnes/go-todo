package login

import (
	"net/http"
	"strings"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
	"github.com/bcomnes/go-todo/pkg/web"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

type routes struct {
	auth     *auth.Service
	sessions *httpx.Sessions
	page     *web.Page
}

type request struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register adds the login page, browser action, and JSON endpoint to mux.
func Register(mux *http.ServeMux, authService *auth.Service, sessions *httpx.Sessions) error {
	page, err := newPage()
	if err != nil {
		return err
	}
	routes := &routes{auth: authService, sessions: sessions, page: page}
	mux.HandleFunc("GET /login", routes.getPage)
	mux.HandleFunc("POST /login", routes.postPage)
	mux.HandleFunc("POST /api/login", routes.postAPI)
	return nil
}

func prepare(input *request) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
}

func (routes *routes) renderError(w http.ResponseWriter, r *http.Request, input request, status int, message string) {
	data := pageData{
		Data:  layout.Data{Title: "Log in"},
		Error: message,
		Email: input.Email,
	}
	httpx.RenderFormError(w, r, status, routes.page, "login-form", data)
}
