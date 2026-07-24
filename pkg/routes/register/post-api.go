package register

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
	if err := validate(input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := routes.auth.CreateUser(r.Context(), auth.Registration{
		Username: input.Username,
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrCapacity):
			w.Header().Set("Retry-After", "1")
			httpx.WriteError(w, http.StatusTooManyRequests, "registration is busy; retry shortly")
		case errors.Is(err, auth.ErrUserExists):
			httpx.WriteError(w, http.StatusConflict, auth.ErrUserExists.Error())
		default:
			httpx.WriteError(w, http.StatusInternalServerError, "failed to create user")
		}
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, user)
}
