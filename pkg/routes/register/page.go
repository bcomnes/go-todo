// Package register owns browser and JSON account-registration routes with their
// shared validation, page, and replaceable form fragment.
package register

import (
	_ "embed"

	"github.com/bcomnes/go-todo/pkg/web"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

//go:embed page.gohtml
var source string

type pageData struct {
	layout.Data
	Error    string
	Notice   string
	Email    string
	Username string
}

func newPage() (*web.Page, error) {
	return web.NewPage("register", source, "register-form")
}
