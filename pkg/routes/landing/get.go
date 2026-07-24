package landing

import (
	"net/http"

	"github.com/bcomnes/go-todo/pkg/httpx"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

func (routes *routes) get(w http.ResponseWriter, r *http.Request) {
	data := pageData{
		Data: layout.Data{
			Title:       "Home",
			CurrentUser: routes.sessions.OptionalUser(r),
		},
		Status:   "Operational",
		StatusOK: true,
		Notice:   "The web interface is ready.",
	}
	if fragment := r.URL.Query().Get("fragment"); fragment != "" {
		if fragment != "health-status" {
			http.NotFound(w, r)
			return
		}
		httpx.RenderFragment(w, http.StatusOK, routes.page, fragment, data)
		return
	}
	httpx.RenderPage(w, http.StatusOK, routes.page, data)
}
