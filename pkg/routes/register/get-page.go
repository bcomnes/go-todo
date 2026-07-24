package register

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
	httpx.RenderPage(w, http.StatusOK, routes.page, pageData{Data: layout.Data{Title: "Register"}})
}
