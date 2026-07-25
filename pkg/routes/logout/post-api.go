package logout

import (
	"context"
	"errors"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/danielgtaylor/huma/v2"
)

type postAPIInput struct{}

type postAPIOutput struct {
	CacheControl string `header:"Cache-Control"`
}

func (routes *routes) postAPI(ctx context.Context, _ *postAPIInput) (*postAPIOutput, error) {
	session, ok := routes.sessions.Current(ctx)
	if !ok {
		return nil, huma.ErrorWithHeaders(
			huma.Error401Unauthorized(auth.ErrUnauthorized.Error()),
			http.Header{"Cache-Control": {"no-store"}},
		)
	}
	if err := routes.auth.Revoke(ctx, session); err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			return nil, huma.ErrorWithHeaders(
				huma.Error401Unauthorized(auth.ErrUnauthorized.Error()),
				http.Header{"Cache-Control": {"no-store"}},
			)
		}
		return nil, huma.ErrorWithHeaders(
			huma.Error500InternalServerError("failed to revoke token"),
			http.Header{"Cache-Control": {"no-store"}},
		)
	}
	return &postAPIOutput{CacheControl: "no-store"}, nil
}
