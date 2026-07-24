package logout

import (
	"errors"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
)

func (routes *routes) postAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Header.Get("Authorization") == "" && !routes.sessions.HasSameOrigin(r) {
		httpx.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	session, ok := routes.sessions.Current(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, auth.ErrUnauthorized.Error())
		return
	}
	if err := routes.auth.Revoke(r.Context(), session); err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			httpx.WriteError(w, http.StatusUnauthorized, auth.ErrUnauthorized.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "failed to revoke token")
		return
	}
	routes.sessions.ClearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
