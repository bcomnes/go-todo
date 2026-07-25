package todos

import (
	"context"
	"errors"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/models"
	todostore "github.com/bcomnes/go-todo/pkg/todos"
	"github.com/danielgtaylor/huma/v2"
)

type listAPIInput struct {
	Limit  int `query:"limit" default:"20" minimum:"1" maximum:"100" doc:"Maximum number of todos to return"`
	Offset int `query:"offset" default:"0" minimum:"0" doc:"Number of todos to skip"`
}

type listAPIOutput struct {
	CacheControl string        `header:"Cache-Control"`
	Body         []models.Todo `nameHint:"TodoList"`
}

type todoIDAPIInput struct {
	ID int64 `path:"id" minimum:"1" doc:"Todo identifier"`
}

type todoAPIOutput struct {
	CacheControl string      `header:"Cache-Control"`
	Body         models.Todo `nameHint:"Todo"`
}

type createAPIRequest struct {
	Task string  `json:"task" minLength:"1" maxLength:"500" doc:"Task description"`
	Done bool    `json:"done,omitempty" doc:"Whether the todo is complete"`
	Note *string `json:"note,omitempty" nullable:"true" maxLength:"5000" doc:"Optional todo notes"`
}

type createAPIInput struct {
	Body createAPIRequest
}

type updateAPIRequest struct {
	Task *string        `json:"task,omitempty" minLength:"1" maxLength:"500" doc:"Replacement task description"`
	Done *bool          `json:"done,omitempty" doc:"Replacement completion state"`
	Note nullableString `json:"note,omitempty" maxLength:"5000" doc:"Replacement notes; null clears the note"`
}

type updateAPIInput struct {
	ID   int64            `path:"id" minimum:"1" doc:"Todo identifier"`
	Body updateAPIRequest `minProperties:"1"`
}

type deleteAPIOutput struct {
	CacheControl string `header:"Cache-Control"`
}

func (input updateAPIRequest) serviceInput() todostore.UpdateInput {
	return todostore.UpdateInput{
		Task:    input.Task,
		Done:    input.Done,
		Note:    input.Note.Value,
		NoteSet: input.Note.Set,
	}
}

func (routes *routes) listAPI(ctx context.Context, input *listAPIInput) (*listAPIOutput, error) {
	userID, err := routes.currentAPIUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := routes.service.List(ctx, userID, input.Limit, input.Offset)
	if err != nil {
		return nil, todoAPIServiceError(err, "failed to list todos")
	}
	return &listAPIOutput{CacheControl: "no-store", Body: items}, nil
}

func (routes *routes) createAPI(ctx context.Context, input *createAPIInput) (*todoAPIOutput, error) {
	userID, err := routes.currentAPIUserID(ctx)
	if err != nil {
		return nil, err
	}
	todo, err := routes.service.Create(ctx, userID, todostore.CreateInput{
		Task: input.Body.Task,
		Done: input.Body.Done,
		Note: input.Body.Note,
	})
	if err != nil {
		return nil, todoAPIServiceError(err, "failed to create todo")
	}
	return &todoAPIOutput{CacheControl: "no-store", Body: todo}, nil
}

func (routes *routes) getAPI(ctx context.Context, input *todoIDAPIInput) (*todoAPIOutput, error) {
	userID, err := routes.currentAPIUserID(ctx)
	if err != nil {
		return nil, err
	}
	todo, err := routes.service.Get(ctx, userID, input.ID)
	if err != nil {
		return nil, todoAPIServiceError(err, "failed to get todo")
	}
	return &todoAPIOutput{CacheControl: "no-store", Body: todo}, nil
}

func (routes *routes) updateAPI(ctx context.Context, input *updateAPIInput) (*todoAPIOutput, error) {
	userID, err := routes.currentAPIUserID(ctx)
	if err != nil {
		return nil, err
	}
	todo, err := routes.service.Update(ctx, userID, input.ID, input.Body.serviceInput())
	if err != nil {
		return nil, todoAPIServiceError(err, "failed to update todo")
	}
	return &todoAPIOutput{CacheControl: "no-store", Body: todo}, nil
}

func (routes *routes) deleteAPI(ctx context.Context, input *todoIDAPIInput) (*deleteAPIOutput, error) {
	userID, err := routes.currentAPIUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := routes.service.Delete(ctx, userID, input.ID); err != nil {
		return nil, todoAPIServiceError(err, "failed to delete todo")
	}
	return &deleteAPIOutput{CacheControl: "no-store"}, nil
}

func (routes *routes) currentAPIUserID(ctx context.Context) (int64, error) {
	session, ok := routes.sessions.Current(ctx)
	if !ok {
		return 0, withNoStore(huma.Error401Unauthorized(auth.ErrUnauthorized.Error()))
	}
	return session.User.ID, nil
}

func todoAPIServiceError(err error, fallback string) error {
	switch {
	case errors.Is(err, todostore.ErrNotFound):
		return withNoStore(huma.Error404NotFound(todostore.ErrNotFound.Error()))
	case todostore.IsValidationError(err):
		return withNoStore(huma.Error400BadRequest(err.Error()))
	default:
		return withNoStore(huma.Error500InternalServerError(fallback))
	}
}

func withNoStore(err error) error {
	return huma.ErrorWithHeaders(err, http.Header{"Cache-Control": {"no-store"}})
}
