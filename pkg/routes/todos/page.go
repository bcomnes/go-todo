package todos

import (
	_ "embed"
	"fmt"

	"github.com/bcomnes/go-todo/pkg/models"
	"github.com/bcomnes/go-todo/pkg/web"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

//go:embed index/page.gohtml
var indexSource string

//go:embed detail/page.gohtml
var detailSource string

//go:embed edit/page.gohtml
var editSource string

const (
	// TodoDetailPagePattern is the registration pattern for a todo permalink.
	TodoDetailPagePattern = "GET /todos/{id}"
	// TodoEditFormPattern is the registration pattern for the progressively enhanced edit form.
	TodoEditFormPattern = "GET /todos/{id}/edit"

	todoListFragment     = "todo-list"
	todoListItemFragment = "todo-list-item"
	todoDetailFragment   = "todo-detail"
	todoEditFormFragment = "todo-edit-form"
	todoEditSavedEvent   = "todoEditSaved"
	todoIndexPageScript  = "/assets/pages/todos/index.js"
	todoDetailPageScript = "/assets/pages/todos/detail.js"
)

type pageData struct {
	layout.Data
	Todos    []models.Todo
	Todo     models.Todo
	EditForm editFormData
	Error    string
	Notice   string
}

type editFormData struct {
	TodoID   int64
	Task     string
	Note     string
	Error    string
	ReturnTo string
	Target   string
}

func newPages() (indexPage, detailPage, editPage *web.Page, err error) {
	indexPage, err = web.NewPage(
		"todos-index",
		indexSource,
		todoListFragment,
		todoListItemFragment,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	detailPage, err = web.NewPage(
		"todos-detail",
		detailSource,
		todoDetailFragment,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	editPage, err = web.NewPage(
		"todos-edit",
		editSource,
		todoEditFormFragment,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return indexPage, detailPage, editPage, nil
}

func todoIndexLayoutData(title string, currentUser *models.User) layout.Data {
	return todoLayoutData(title, currentUser, todoIndexPageScript)
}

func todoDetailLayoutData(title string, currentUser *models.User) layout.Data {
	return todoLayoutData(title, currentUser, todoDetailPageScript)
}

func todoEditLayoutData(title string, currentUser *models.User) layout.Data {
	return todoLayoutData(title, currentUser)
}

func todoLayoutData(title string, currentUser *models.User, scripts ...string) layout.Data {
	return layout.Data{
		Title:       title,
		CurrentUser: currentUser,
		Scripts:     scripts,
	}
}

func newEditFormData(todo models.Todo, returnTo string) editFormData {
	note := ""
	if todo.Note != nil {
		note = *todo.Note
	}
	return editFormData{
		TodoID:   todo.ID,
		Task:     todo.Task,
		Note:     note,
		ReturnTo: returnTo,
		Target:   editTarget(returnTo, todo.ID),
	}
}

func editTarget(returnTo string, todoID int64) string {
	if returnTo == fmt.Sprintf("/todos/%d", todoID) {
		return "#todo-detail"
	}
	return fmt.Sprintf("#todo-%d", todoID)
}
