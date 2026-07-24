package account

import (
	"net/http"

	"github.com/bcomnes/go-todo/pkg/httpx"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

func (routes *routes) getPage(w http.ResponseWriter, r *http.Request) {
	session, ok := routes.sessions.Current(r.Context())
	if !ok {
		httpx.Redirect(w, r, "/login")
		return
	}
	data := pageData{
		Data: layout.Data{
			Title:       "Account",
			CurrentUser: &session.User,
		},
		Notice: "Account data loaded from the authenticated session.",
	}
	if fragment := r.URL.Query().Get("fragment"); fragment != "" {
		if fragment != "account-card" {
			http.NotFound(w, r)
			return
		}
		httpx.RenderFragment(w, http.StatusOK, routes.page, fragment, data)
		return
	}
	httpx.RenderPage(w, http.StatusOK, routes.page, data)
}
