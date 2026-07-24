package todos

import (
	"errors"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/httpx"
	todostore "github.com/bcomnes/go-todo/pkg/todos"
)

type createAPIRequest struct {
	Task string  `json:"task"`
	Done bool    `json:"done"`
	Note *string `json:"note"`
}

type updateAPIRequest struct {
	Task *string        `json:"task"`
	Done *bool          `json:"done"`
	Note nullableString `json:"note"`
}

func (input updateAPIRequest) serviceInput() todostore.UpdateInput {
	return todostore.UpdateInput{
		Task:    input.Task,
		Done:    input.Done,
		Note:    input.Note.Value,
		NoteSet: input.Note.Set,
	}
}

func (routes *routes) listAPI(w http.ResponseWriter, r *http.Request) {
	userID, ok := routes.currentAPIUserID(w, r)
	if !ok {
		return
	}
	page, err := parsePagination(r.URL.Query())
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := routes.service.List(r.Context(), userID, page.Limit, page.Offset)
	if err != nil {
		writeAPIServiceError(w, err, "failed to list todos")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, items)
}

func (routes *routes) createAPI(w http.ResponseWriter, r *http.Request) {
	userID, ok := routes.currentAPIUserID(w, r)
	if !ok {
		return
	}
	var input createAPIRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteDecodeError(w, err)
		return
	}
	todo, err := routes.service.Create(r.Context(), userID, todostore.CreateInput{
		Task: input.Task,
		Done: input.Done,
		Note: input.Note,
	})
	if err != nil {
		writeAPIServiceError(w, err, "failed to create todo")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, todo)
}

func (routes *routes) getAPI(w http.ResponseWriter, r *http.Request) {
	userID, ok := routes.currentAPIUserID(w, r)
	if !ok {
		return
	}
	id, err := parseTodoID(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	todo, err := routes.service.Get(r.Context(), userID, id)
	if err != nil {
		writeAPIServiceError(w, err, "failed to get todo")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, todo)
}

func (routes *routes) updateAPI(w http.ResponseWriter, r *http.Request) {
	userID, ok := routes.currentAPIUserID(w, r)
	if !ok {
		return
	}
	id, err := parseTodoID(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	var input updateAPIRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteDecodeError(w, err)
		return
	}
	todo, err := routes.service.Update(r.Context(), userID, id, input.serviceInput())
	if err != nil {
		writeAPIServiceError(w, err, "failed to update todo")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, todo)
}

func (routes *routes) deleteAPI(w http.ResponseWriter, r *http.Request) {
	userID, ok := routes.currentAPIUserID(w, r)
	if !ok {
		return
	}
	id, err := parseTodoID(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := routes.service.Delete(r.Context(), userID, id); err != nil {
		writeAPIServiceError(w, err, "failed to delete todo")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (routes *routes) currentAPIUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	session, ok := routes.sessions.Current(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, auth.ErrUnauthorized.Error())
		return 0, false
	}
	return session.User.ID, true
}

func writeAPIServiceError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, todostore.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, todostore.ErrNotFound.Error())
	case todostore.IsValidationError(err):
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, fallback)
	}
}
