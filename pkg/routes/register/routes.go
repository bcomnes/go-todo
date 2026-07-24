package register

import (
	"errors"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
	"github.com/bcomnes/go-todo/pkg/security"
	"github.com/bcomnes/go-todo/pkg/web"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{2,99}$`)

type routes struct {
	auth     *auth.Service
	sessions *httpx.Sessions
	page     *web.Page
}

type request struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register adds the registration page, browser action, and JSON endpoint to mux.
func Register(mux *http.ServeMux, authService *auth.Service, sessions *httpx.Sessions) error {
	page, err := newPage()
	if err != nil {
		return err
	}
	routes := &routes{auth: authService, sessions: sessions, page: page}
	mux.HandleFunc("GET /register", routes.getPage)
	mux.HandleFunc("POST /register", routes.postPage)
	mux.HandleFunc("POST /api/register", routes.postAPI)
	return nil
}

func prepare(input *request) {
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
}

func validate(input request) error {
	if !usernamePattern.MatchString(input.Username) {
		return errors.New("username must be 3-100 characters and contain only letters, numbers, dots, dashes, or underscores")
	}
	if utf8.RuneCountInString(input.Email) > 320 {
		return errors.New("email is too long")
	}
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email {
		return errors.New("email is invalid")
	}
	if len(input.Password) < 12 {
		return errors.New("password must be at least 12 bytes")
	}
	if len(input.Password) > security.MaxPasswordLen {
		return errors.New("password must be at most 72 bytes")
	}
	return nil
}

func (routes *routes) renderError(w http.ResponseWriter, r *http.Request, input request, status int, message string) {
	data := pageData{
		Data:     layout.Data{Title: "Register"},
		Error:    message,
		Email:    input.Email,
		Username: input.Username,
	}
	httpx.RenderFormError(w, r, status, routes.page, "register-form", data)
}
