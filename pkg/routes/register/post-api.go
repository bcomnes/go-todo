package register

import (
	"context"
	"errors"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/danielgtaylor/huma/v2"
)

type postAPIInput struct {
	Body registerRequest
}

type postAPIOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         auth.AuthResult
}

func (routes *routes) postAPI(ctx context.Context, input *postAPIInput) (*postAPIOutput, error) {
	prepare(&input.Body)
	if err := validate(input.Body); err != nil {
		return nil, huma.ErrorWithHeaders(
			huma.Error400BadRequest(err.Error()),
			http.Header{"Cache-Control": {"no-store"}},
		)
	}
	result, err := routes.auth.Register(ctx, auth.Registration{
		Username: input.Body.Username,
		Email:    input.Body.Email,
		Password: input.Body.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrCapacity):
			return nil, huma.ErrorWithHeaders(
				huma.Error429TooManyRequests("registration is busy; retry shortly"),
				http.Header{
					"Cache-Control": {"no-store"},
					"Retry-After":   {"1"},
				},
			)
		case errors.Is(err, auth.ErrUserExists):
			return nil, huma.ErrorWithHeaders(
				huma.Error409Conflict(auth.ErrUserExists.Error()),
				http.Header{"Cache-Control": {"no-store"}},
			)
		default:
			return nil, huma.ErrorWithHeaders(
				huma.Error500InternalServerError("failed to create user"),
				http.Header{"Cache-Control": {"no-store"}},
			)
		}
	}
	return &postAPIOutput{CacheControl: "no-store", Body: result}, nil
}
