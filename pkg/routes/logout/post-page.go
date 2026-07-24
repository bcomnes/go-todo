package logout

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
	session, ok := routes.sessions.Current(r.Context())
	if !ok {
		routes.sessions.ClearCookie(w)
		httpx.Redirect(w, r, "/login")
		return
	}
	if err := routes.auth.Revoke(r.Context(), session); err != nil && !errors.Is(err, auth.ErrUnauthorized) {
		http.Error(w, "failed to revoke token", http.StatusInternalServerError)
		return
	}
	routes.sessions.ClearCookie(w)
	httpx.Redirect(w, r, "/")
}
