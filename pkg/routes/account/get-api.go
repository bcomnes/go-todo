package account

import (
	"context"
	"net/http"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/models"
	"github.com/danielgtaylor/huma/v2"
)

type getAPIInput struct{}

type getAPIOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         models.User
}

func (routes *routes) getAPI(ctx context.Context, _ *getAPIInput) (*getAPIOutput, error) {
	session, ok := routes.sessions.Current(ctx)
	if !ok {
		return nil, huma.ErrorWithHeaders(
			huma.Error401Unauthorized(auth.ErrUnauthorized.Error()),
			http.Header{"Cache-Control": {"no-store"}},
		)
	}
	return &getAPIOutput{CacheControl: "no-store", Body: session.User}, nil
}
