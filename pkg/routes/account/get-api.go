package account

import (
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
)

func (routes *routes) getAPI(w http.ResponseWriter, r *http.Request) {
	session, ok := routes.sessions.Current(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, auth.ErrUnauthorized.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session.User)
}
