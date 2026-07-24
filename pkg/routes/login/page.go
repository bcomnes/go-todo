// Package login owns browser and JSON login routes with their shared operation,
// page, and replaceable form fragment.
package login

import (
	_ "embed"

	"github.com/bcomnes/go-todo/pkg/web"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

//go:embed page.html
var source string

type pageData struct {
	layout.Data
	Error  string
	Notice string
	Email  string
}

func newPage() (*web.Page, error) {
	return web.NewPage("login", source, "login-form")
}
