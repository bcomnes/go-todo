package todos

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
	todostore "github.com/bcomnes/go-todo/pkg/todos"
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
	httpx.RenderPage(w, http.StatusOK, routes.indexPage, pageData{
		Data:  todoIndexLayoutData("Todos", &session.User),
		Todos: items,
	})
}

// GetDetailPage renders the full page for an owner-scoped todo permalink.
// Register it with TodoDetailPagePattern and sessions.RequirePage.
func (routes *routes) GetDetailPage(w http.ResponseWriter, r *http.Request) {
	session, ok := routes.currentPageSession(w, r)
	if !ok {
		return
	}
	id, err := parseTodoID(r.PathValue("id"))
	if err != nil {
		message, status := publicPageError(err)
		http.Error(w, message, status)
		return
	}
	todo, err := routes.service.Get(r.Context(), session.User.ID, id)
	if err != nil {
		message, status := publicPageError(err)
		http.Error(w, message, status)
		return
	}
	httpx.RenderPage(w, http.StatusOK, routes.detailPage, pageData{
		Data: todoDetailLayoutData(todo.Task, &session.User),
		Todo: todo,
	})
}

// GetEditForm renders the edit-form fragment for HTMX and a complete, usable
// edit page for ordinary browser navigation.
// Register it with TodoEditFormPattern and sessions.RequirePage.
func (routes *routes) GetEditForm(w http.ResponseWriter, r *http.Request) {
	session, ok := routes.currentPageSession(w, r)
	if !ok {
		return
	}
	id, err := parseTodoID(r.PathValue("id"))
	if err != nil {
		message, status := publicPageError(err)
		http.Error(w, message, status)
		return
	}
	todo, err := routes.service.Get(r.Context(), session.User.ID, id)
	if err != nil {
		message, status := publicPageError(err)
		http.Error(w, message, status)
		return
	}
	requestedReturn := r.URL.Query().Get("return_to")
	if requestedReturn == "" {
		requestedReturn = fmt.Sprintf("/todos/%d", id)
	}
	returnTo := editReturnTo(requestedReturn, id)
	editForm := newEditFormData(todo, returnTo)
	if httpx.IsHTMX(r) {
		httpx.RenderFragment(w, http.StatusOK, routes.editPage, todoEditFormFragment, editForm)
		return
	}
	httpx.RenderPage(w, http.StatusOK, routes.editPage, pageData{
		Data:     todoEditLayoutData("Edit "+todo.Task, &session.User),
		Todo:     todo,
		EditForm: editForm,
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
		routes.respondEditError(w, r, session, id, editFormData{
			TodoID:   id,
			Error:    "invalid form submission",
			ReturnTo: "/todos",
			Target:   editTarget("/todos", id),
		}, errors.New("invalid form submission"))
		return
	}

	task := r.PostForm.Get("task")
	note := r.PostForm.Get("note")
	returnTo := editReturnTo(r.PostForm.Get("return_to"), id)
	updated, err := routes.service.Update(r.Context(), session.User.ID, id, todostore.UpdateInput{
		Task:    &task,
		Note:    noteFromForm(note),
		NoteSet: true,
	})
	if err != nil {
		message, _ := publicPageError(err)
		routes.respondEditError(w, r, session, id, editFormData{
			TodoID:   id,
			Task:     task,
			Note:     note,
			Error:    message,
			ReturnTo: returnTo,
			Target:   editTarget(returnTo, id),
		}, err)
		return
	}
	if !httpx.IsHTMX(r) {
		httpx.Redirect(w, r, returnTo)
		return
	}

	w.Header().Set("HX-Trigger", todoEditSavedEvent)
	if returnTo == "/todos" {
		httpx.RenderFragment(w, http.StatusOK, routes.indexPage, todoListItemFragment, updated)
		return
	}
	httpx.RenderFragment(w, http.StatusOK, routes.detailPage, todoDetailFragment, pageData{Todo: updated})
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

func (routes *routes) respondEditError(w http.ResponseWriter, r *http.Request, session auth.Session, id int64, form editFormData, operationErr error) {
	_, status := publicPageError(operationErr)
	if httpx.IsHTMX(r) {
		w.Header().Set("HX-Retarget", "#todo-edit-content")
		w.Header().Set("HX-Reswap", "innerHTML")
		httpx.RenderFragment(w, status, routes.editPage, todoEditFormFragment, form)
		return
	}

	todo, err := routes.service.Get(r.Context(), session.User.ID, id)
	if err != nil {
		http.Error(w, "failed to reload todo", http.StatusInternalServerError)
		return
	}
	httpx.RenderPage(w, status, routes.editPage, pageData{
		Data:     todoEditLayoutData("Edit "+todo.Task, &session.User),
		Todo:     todo,
		EditForm: form,
	})
}

func editReturnTo(value string, id int64) string {
	detail := fmt.Sprintf("/todos/%d", id)
	if value == detail || value == "/todos" {
		return value
	}
	return "/todos"
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
		httpx.Redirect(w, r, editReturnTo(r.PostForm.Get("return_to"), pathTodoID(r)))
		return
	}
	items, err := routes.service.List(r.Context(), session.User.ID, defaultListLimit, 0)
	if err != nil {
		http.Error(w, "failed to refresh todos", http.StatusInternalServerError)
		return
	}
	data := pageData{
		Data:   todoIndexLayoutData("Todos", &session.User),
		Todos:  items,
		Notice: notice,
	}
	status := http.StatusOK
	if operationErr != nil {
		data.Notice = ""
		data.Error, status = publicPageError(operationErr)
	}
	if httpx.IsHTMX(r) {
		httpx.RenderFragment(w, status, routes.indexPage, todoListFragment, data)
		return
	}
	httpx.RenderPage(w, status, routes.indexPage, data)
}

func pathTodoID(r *http.Request) int64 {
	id, _ := parseTodoID(r.PathValue("id"))
	return id
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
