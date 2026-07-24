package register

import (
	"errors"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
)

func (routes *routes) postPage(w http.ResponseWriter, r *http.Request) {
	if !routes.sessions.HasSameOrigin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := httpx.ParseForm(w, r); err != nil {
		routes.renderError(w, r, request{}, http.StatusBadRequest, "invalid form submission")
		return
	}
	input := request{
		Username: r.PostForm.Get("username"),
		Email:    r.PostForm.Get("email"),
		Password: r.PostForm.Get("password"),
	}
	prepare(&input)
	if err := validate(input); err != nil {
		routes.renderError(w, r, input, http.StatusBadRequest, err.Error())
		return
	}
	result, err := routes.auth.Register(r.Context(), auth.Registration{
		Username: input.Username,
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrCapacity):
			w.Header().Set("Retry-After", "1")
			routes.renderError(w, r, input, http.StatusTooManyRequests, "registration is busy; retry shortly")
		case errors.Is(err, auth.ErrUserExists):
			routes.renderError(w, r, input, http.StatusConflict, auth.ErrUserExists.Error())
		default:
			routes.renderError(w, r, input, http.StatusInternalServerError, "failed to create user")
		}
		return
	}
	routes.sessions.SetCookie(w, result.Token, result.ExpiresAt)
	httpx.Redirect(w, r, "/todos")
}
