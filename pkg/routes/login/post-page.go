package login

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
	input := request{Email: r.PostForm.Get("email"), Password: r.PostForm.Get("password")}
	prepare(&input)
	result, err := routes.auth.Login(r.Context(), auth.Credentials{Email: input.Email, Password: input.Password})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrCapacity):
			w.Header().Set("Retry-After", "1")
			routes.renderError(w, r, input, http.StatusTooManyRequests, "authentication is busy; retry shortly")
		case errors.Is(err, auth.ErrInvalidCredentials):
			routes.renderError(w, r, input, http.StatusUnauthorized, auth.ErrInvalidCredentials.Error())
		case errors.Is(err, auth.ErrUnavailable):
			routes.renderError(w, r, input, http.StatusServiceUnavailable, auth.ErrUnavailable.Error())
		default:
			routes.renderError(w, r, input, http.StatusInternalServerError, auth.ErrTokenCreation.Error())
		}
		return
	}
	routes.sessions.SetCookie(w, result.Token, result.ExpiresAt)
	httpx.Redirect(w, r, "/todos")
}
