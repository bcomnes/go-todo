package login

import (
	"net/http"

	"github.com/bcomnes/go-todo/pkg/httpx"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

func (routes *routes) getPage(w http.ResponseWriter, r *http.Request) {
	if routes.sessions.OptionalUser(r) != nil {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}
	data := pageData{Data: layout.Data{Title: "Log in"}}
	if r.URL.Query().Get("registered") == "1" {
		data.Notice = "Account created. You can log in now."
	}
	httpx.RenderPage(w, http.StatusOK, routes.page, data)
}
