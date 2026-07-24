package todos

import (
	_ "embed"

	"github.com/bcomnes/go-todo/pkg/models"
	"github.com/bcomnes/go-todo/pkg/web"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

//go:embed page.html
var source string

type pageData struct {
	layout.Data
	Todos  []models.Todo
	Error  string
	Notice string
}

func newPage() (*web.Page, error) {
	return web.NewPage("todos", source, todoListFragment)
}
