package login

import (
	"context"
	"errors"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/danielgtaylor/huma/v2"
)

type postAPIInput struct {
	Body loginRequest
}

type postAPIOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         auth.AuthResult
}

func (routes *routes) postAPI(ctx context.Context, input *postAPIInput) (*postAPIOutput, error) {
	prepare(&input.Body)
	result, err := routes.auth.Login(ctx, auth.Credentials{
		Email:    input.Body.Email,
		Password: input.Body.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrCapacity):
			return nil, huma.ErrorWithHeaders(
				huma.Error429TooManyRequests("authentication is busy; retry shortly"),
				http.Header{
					"Cache-Control": {"no-store"},
					"Retry-After":   {"1"},
				},
			)
		case errors.Is(err, auth.ErrInvalidCredentials):
			return nil, huma.ErrorWithHeaders(
				huma.Error401Unauthorized(auth.ErrInvalidCredentials.Error()),
				http.Header{"Cache-Control": {"no-store"}},
			)
		case errors.Is(err, auth.ErrUnavailable):
			return nil, huma.ErrorWithHeaders(
				huma.Error503ServiceUnavailable(auth.ErrUnavailable.Error()),
				http.Header{"Cache-Control": {"no-store"}},
			)
		default:
			return nil, huma.ErrorWithHeaders(
				huma.Error500InternalServerError(auth.ErrTokenCreation.Error()),
				http.Header{"Cache-Control": {"no-store"}},
			)
		}
	}
	return &postAPIOutput{CacheControl: "no-store", Body: result}, nil
}
