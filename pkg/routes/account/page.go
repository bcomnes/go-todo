// Package account owns authenticated browser and JSON account routes with their
// page and refreshable account-card fragment.
package account

import (
	_ "embed"

	"github.com/bcomnes/go-todo/pkg/web"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

//go:embed page.gohtml
var source string

type pageData struct {
	layout.Data
	Error  string
	Notice string
}

func newPage() (*web.Page, error) {
	return web.NewPage("account", source, "account-card")
}
