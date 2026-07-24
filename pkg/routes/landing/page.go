// Package landing owns the public landing route, page, and HTMX fragment.
package landing

import (
	_ "embed"

	"github.com/bcomnes/go-todo/pkg/web"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

//go:embed page.gohtml
var source string

type pageData struct {
	layout.Data
	Status   string
	StatusOK bool
	Notice   string
	Error    string
}

func newPage() (*web.Page, error) {
	return web.NewPage("landing", source, "health-status")
}
