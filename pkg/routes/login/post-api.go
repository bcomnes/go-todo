package login

import (
	"errors"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
)

func (routes *routes) postAPI(w http.ResponseWriter, r *http.Request) {
	var input request
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteDecodeError(w, err)
		return
	}
	prepare(&input)
	result, err := routes.auth.Login(r.Context(), auth.Credentials{Email: input.Email, Password: input.Password})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrCapacity):
			w.Header().Set("Retry-After", "1")
			httpx.WriteError(w, http.StatusTooManyRequests, "authentication is busy; retry shortly")
		case errors.Is(err, auth.ErrInvalidCredentials):
			httpx.WriteError(w, http.StatusUnauthorized, auth.ErrInvalidCredentials.Error())
		case errors.Is(err, auth.ErrUnavailable):
			httpx.WriteError(w, http.StatusServiceUnavailable, auth.ErrUnavailable.Error())
		default:
			httpx.WriteError(w, http.StatusInternalServerError, auth.ErrTokenCreation.Error())
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}
