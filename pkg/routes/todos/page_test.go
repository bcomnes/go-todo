package todos

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/bcomnes/go-todo/pkg/models"
)

func TestTodoListFragmentRendersRequiredControls(t *testing.T) {
	page, err := newPage()
	if err != nil {
		t.Fatalf("newPage: %v", err)
	}
	note := "Important context"
	data := pageData{Todos: []models.Todo{{
		ID:        7,
		Task:      "Write tests",
		Done:      true,
		Note:      &note,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}}}
	var output bytes.Buffer
	if err := page.RenderFragment(&output, todoListFragment, data); err != nil {
		t.Fatalf("RenderFragment: %v", err)
	}

	html := output.String()
	for _, required := range []string{
		`id="todo-list"`,
		`Completed`,
		`Important context`,
		`action="/todos/7/toggle"`,
		`action="/todos/7/delete"`,
		`action="/todos/7"`,
		`hx-target="#todo-list"`,
		`hx-swap="outerHTML"`,
	} {
		if !strings.Contains(html, required) {
			t.Errorf("fragment does not contain %q", required)
		}
	}
}
