package todos

import (
	"errors"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
	todostore "github.com/bcomnes/go-todo/pkg/todos"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

func (routes *routes) getPage(w http.ResponseWriter, r *http.Request) {
	session, ok := routes.sessions.Current(r.Context())
	if !ok {
		httpx.Redirect(w, r, "/login")
		return
	}
	items, err := routes.service.List(r.Context(), session.User.ID, defaultListLimit, 0)
	if err != nil {
		http.Error(w, "failed to load todos", http.StatusInternalServerError)
		return
	}
	httpx.RenderPage(w, http.StatusOK, routes.page, pageData{
		Data:  layout.Data{Title: "Todos", CurrentUser: &session.User},
		Todos: items,
	})
}

func (routes *routes) createPage(w http.ResponseWriter, r *http.Request) {
	session, ok := routes.currentPageSession(w, r)
	if !ok {
		return
	}
	if err := httpx.ParseForm(w, r); err != nil {
		routes.respondPageMutation(w, r, session, errors.New("invalid form submission"), "")
		return
	}
	_, err := routes.service.Create(r.Context(), session.User.ID, todostore.CreateInput{
		Task: r.PostForm.Get("task"),
		Note: noteFromForm(r.PostForm.Get("note")),
	})
	routes.respondPageMutation(w, r, session, err, "Todo created.")
}

func (routes *routes) updatePage(w http.ResponseWriter, r *http.Request) {
	session, ok := routes.currentPageSession(w, r)
	if !ok {
		return
	}
	id, err := parseTodoID(r.PathValue("id"))
	if err != nil {
		routes.respondPageMutation(w, r, session, err, "")
		return
	}
	if err := httpx.ParseForm(w, r); err != nil {
		routes.respondPageMutation(w, r, session, errors.New("invalid form submission"), "")
		return
	}
	task := r.PostForm.Get("task")
	_, err = routes.service.Update(r.Context(), session.User.ID, id, todostore.UpdateInput{
		Task:    &task,
		Note:    noteFromForm(r.PostForm.Get("note")),
		NoteSet: true,
	})
	routes.respondPageMutation(w, r, session, err, "Todo updated.")
}

func (routes *routes) togglePage(w http.ResponseWriter, r *http.Request) {
	session, ok := routes.currentPageSession(w, r)
	if !ok {
		return
	}
	id, err := parseTodoID(r.PathValue("id"))
	if err == nil {
		err = httpx.ParseForm(w, r)
		if err != nil {
			err = errors.New("invalid form submission")
		} else {
			_, err = routes.service.Toggle(r.Context(), session.User.ID, id)
		}
	}
	routes.respondPageMutation(w, r, session, err, "Todo completion updated.")
}

func (routes *routes) deletePage(w http.ResponseWriter, r *http.Request) {
	session, ok := routes.currentPageSession(w, r)
	if !ok {
		return
	}
	id, err := parseTodoID(r.PathValue("id"))
	if err == nil {
		err = httpx.ParseForm(w, r)
		if err != nil {
			err = errors.New("invalid form submission")
		} else {
			err = routes.service.Delete(r.Context(), session.User.ID, id)
		}
	}
	routes.respondPageMutation(w, r, session, err, "Todo deleted.")
}

func (routes *routes) currentPageSession(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	session, ok := routes.sessions.Current(r.Context())
	if !ok {
		httpx.Redirect(w, r, "/login")
		return auth.Session{}, false
	}
	return session, true
}

func (routes *routes) respondPageMutation(w http.ResponseWriter, r *http.Request, session auth.Session, operationErr error, notice string) {
	if operationErr == nil && !httpx.IsHTMX(r) {
		httpx.Redirect(w, r, "/todos")
		return
	}
	items, err := routes.service.List(r.Context(), session.User.ID, defaultListLimit, 0)
	if err != nil {
		http.Error(w, "failed to refresh todos", http.StatusInternalServerError)
		return
	}
	data := pageData{
		Data:   layout.Data{Title: "Todos", CurrentUser: &session.User},
		Todos:  items,
		Notice: notice,
	}
	status := http.StatusOK
	if operationErr != nil {
		data.Notice = ""
		data.Error, status = publicPageError(operationErr)
	}
	if httpx.IsHTMX(r) {
		httpx.RenderFragment(w, status, routes.page, todoListFragment, data)
		return
	}
	httpx.RenderPage(w, status, routes.page, data)
}

func publicPageError(err error) (string, int) {
	switch {
	case errors.Is(err, todostore.ErrNotFound):
		return err.Error(), http.StatusNotFound
	case todostore.IsValidationError(err), errors.Is(err, errInvalidTodoID), err.Error() == "invalid form submission":
		return err.Error(), http.StatusBadRequest
	default:
		return "todo operation failed", http.StatusInternalServerError
	}
}
