package todos

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/models"
)

func TestTodoIndexPageMarksCreateFormAndLoadsItsClient(t *testing.T) {
	indexPage, _, _, err := newPages()
	if err != nil {
		t.Fatalf("newPages: %v", err)
	}
	var output bytes.Buffer
	if err := indexPage.RenderPage(&output, pageData{Data: todoIndexLayoutData("Todos", nil)}); err != nil {
		t.Fatalf("RenderPage: %v", err)
	}
	html := output.String()
	if !strings.Contains(html, `id="todo-create-form"`) ||
		!strings.Contains(html, `data-reset-on-success`) {
		t.Fatal("todo create form is not marked to reset after a successful HTMX request")
	}
	if !strings.Contains(html, `id="todo-edit-dialog"`) ||
		!strings.Contains(html, `id="todo-edit-content"`) {
		t.Fatal("todo index page does not contain the reusable edit dialog")
	}
	if !strings.Contains(html, `src="/assets/pages/todos/index.js"`) {
		t.Fatal("todo index page does not load its page client entry")
	}
	if strings.Contains(html, `/assets/pages/todos/detail.js`) {
		t.Fatal("todo index page loads the detail page client entry")
	}
}

func TestTodoListFragmentRendersRequiredControls(t *testing.T) {
	indexPage, _, _, err := newPages()
	if err != nil {
		t.Fatalf("newPages: %v", err)
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
	if err := indexPage.RenderFragment(&output, todoListFragment, data); err != nil {
		t.Fatalf("RenderFragment: %v", err)
	}

	html := output.String()
	for _, required := range []string{
		`id="todo-list"`,
		`Completed`,
		`Important context`,
		`action="/todos/7/toggle"`,
		`action="/todos/7/delete"`,
		`href="/todos/7"`,
		`hx-get="/todos/7/edit?return_to=%2Ftodos"`,
		`hx-target="#todo-edit-content"`,
		`hx-target="#todo-list"`,
		`hx-swap="outerHTML"`,
	} {
		if !strings.Contains(html, required) {
			t.Errorf("fragment does not contain %q", required)
		}
	}
}

func TestTodoDetailPageRendersPermalinkContentAndItsClient(t *testing.T) {
	_, detailPage, _, err := newPages()
	if err != nil {
		t.Fatalf("newPages: %v", err)
	}
	note := "Permalink context"
	data := pageData{
		Data: todoDetailLayoutData("Review detail page", nil),
		Todo: models.Todo{
			ID:   9,
			Task: "Review detail page",
			Note: &note,
		},
	}
	var output bytes.Buffer
	if err := detailPage.RenderPage(&output, data); err != nil {
		t.Fatalf("RenderPage: %v", err)
	}

	html := output.String()
	for _, required := range []string{
		`id="todo-detail-title"`,
		`Review detail page`,
		`Permalink context`,
		`href="/todos/9/edit"`,
		`hx-get="/todos/9/edit"`,
		`id="todo-edit-dialog"`,
		`src="/assets/pages/todos/detail.js"`,
	} {
		if !strings.Contains(html, required) {
			t.Errorf("detail page does not contain %q", required)
		}
	}
	if strings.Contains(html, `/assets/pages/todos/index.js`) {
		t.Fatal("todo detail page loads the index page client entry")
	}
}

func TestTodoEditFormPreservesSubmittedStateAndTargetsOwningFragment(t *testing.T) {
	_, _, editPage, err := newPages()
	if err != nil {
		t.Fatalf("newPages: %v", err)
	}
	data := editFormData{
		TodoID:   12,
		Task:     "Submitted & invalid",
		Note:     "Keep this note",
		Error:    "task is required",
		ReturnTo: "/todos/12",
		Target:   "#todo-detail",
	}
	var output bytes.Buffer
	if err := editPage.RenderFragment(&output, todoEditFormFragment, data); err != nil {
		t.Fatalf("RenderFragment: %v", err)
	}

	html := output.String()
	for _, required := range []string{
		`action="/todos/12"`,
		`value="Submitted &amp; invalid"`,
		`Keep this note`,
		`role="alert"`,
		`aria-invalid="true"`,
		`aria-describedby="todo-edit-error"`,
		`value="/todos/12"`,
		`hx-target="#todo-detail"`,
		`hx-swap="outerHTML"`,
		`data-dialog-close`,
	} {
		if !strings.Contains(html, required) {
			t.Errorf("edit form does not contain %q", required)
		}
	}
}

func TestTodoEditValidationFailureRetargetsDialogContent(t *testing.T) {
	_, _, editPage, err := newPages()
	if err != nil {
		t.Fatalf("newPages: %v", err)
	}
	routes := routes{editPage: editPage}
	request := httptest.NewRequest(http.MethodPost, "/todos/12", nil)
	request.Header.Set("HX-Request", "true")
	response := httptest.NewRecorder()

	routes.respondEditError(response, request, auth.Session{}, 12, editFormData{
		TodoID:   12,
		Task:     "Submitted task",
		Note:     "Submitted note",
		Error:    "task is required",
		ReturnTo: "/todos/12",
	}, errors.New("invalid form submission"))

	if response.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if got := response.Header().Get("HX-Retarget"); got != "#todo-edit-content" {
		t.Errorf("HX-Retarget = %q, want %q", got, "#todo-edit-content")
	}
	if got := response.Header().Get("HX-Reswap"); got != "innerHTML" {
		t.Errorf("HX-Reswap = %q, want %q", got, "innerHTML")
	}
	for _, required := range []string{"Submitted task", "Submitted note", `role="alert"`} {
		if !strings.Contains(response.Body.String(), required) {
			t.Errorf("validation response does not contain %q", required)
		}
	}
}

func TestTodoEditPageWorksWithoutJavaScript(t *testing.T) {
	_, _, editPage, err := newPages()
	if err != nil {
		t.Fatalf("newPages: %v", err)
	}
	todo := models.Todo{ID: 15, Task: "Edit without JavaScript"}
	var output bytes.Buffer
	if err := editPage.RenderPage(&output, pageData{
		Data:     todoEditLayoutData("Edit "+todo.Task, nil),
		Todo:     todo,
		EditForm: newEditFormData(todo, "/todos/15"),
	}); err != nil {
		t.Fatalf("RenderPage: %v", err)
	}

	html := output.String()
	for _, required := range []string{
		`<!doctype html>`,
		`id="todo-edit-title"`,
		`action="/todos/15"`,
		`href="/todos/15"`,
	} {
		if !strings.Contains(strings.ToLower(html), strings.ToLower(required)) {
			t.Errorf("edit page does not contain %q", required)
		}
	}
	if strings.Contains(html, `/assets/pages/todos/`) {
		t.Fatal("standalone edit page unexpectedly loads a page client entry")
	}
}

func TestTodoEditSuccessFragmentsReplaceTheirStateDependentRegion(t *testing.T) {
	indexPage, detailPage, _, err := newPages()
	if err != nil {
		t.Fatalf("newPages: %v", err)
	}
	todo := models.Todo{ID: 18, Task: "Updated task"}

	for _, test := range []struct {
		name     string
		fragment string
		render   func(*bytes.Buffer) error
		prefix   string
		contains string
	}{
		{
			name:     "index",
			fragment: todoListItemFragment,
			render: func(output *bytes.Buffer) error {
				return indexPage.RenderFragment(output, todoListItemFragment, todo)
			},
			prefix:   `<article id="todo-18"`,
			contains: `href="/todos/18"`,
		},
		{
			name:     "detail",
			fragment: todoDetailFragment,
			render: func(output *bytes.Buffer) error {
				return detailPage.RenderFragment(output, todoDetailFragment, pageData{Todo: todo})
			},
			prefix:   `<section id="todo-detail"`,
			contains: `<h1 id="todo-detail-title">Updated task</h1>`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := test.render(&output); err != nil {
				t.Fatalf("RenderFragment(%q): %v", test.fragment, err)
			}
			html := strings.TrimSpace(output.String())
			if !strings.HasPrefix(html, test.prefix) {
				t.Fatalf("fragment does not replace its owning region: %s", html)
			}
			if !strings.Contains(html, test.contains) {
				t.Errorf("fragment does not contain %q", test.contains)
			}
			if strings.Contains(html, "hx-swap-oob") {
				t.Fatal("fragment unexpectedly uses an out-of-band swap")
			}
		})
	}
}

func TestTodoPagesOnlyAllowOwnedFragments(t *testing.T) {
	indexPage, detailPage, editPage, err := newPages()
	if err != nil {
		t.Fatalf("newPages: %v", err)
	}
	for _, test := range []struct {
		name     string
		fragment string
		render   func() error
	}{
		{
			name:     "index rejects edit form",
			fragment: todoEditFormFragment,
			render: func() error {
				return indexPage.RenderFragment(&bytes.Buffer{}, todoEditFormFragment, editFormData{})
			},
		},
		{
			name:     "detail rejects list item",
			fragment: todoListItemFragment,
			render: func() error {
				return detailPage.RenderFragment(&bytes.Buffer{}, todoListItemFragment, models.Todo{})
			},
		},
		{
			name:     "edit rejects detail item",
			fragment: todoDetailFragment,
			render: func() error {
				return editPage.RenderFragment(&bytes.Buffer{}, todoDetailFragment, pageData{})
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.render(); err == nil {
				t.Fatalf("RenderFragment(%q) error = nil", test.fragment)
			}
		})
	}
}

func TestEditReturnToOnlyAllowsTodoPages(t *testing.T) {
	for _, test := range []struct {
		value string
		want  string
	}{
		{value: "/todos", want: "/todos"},
		{value: "/todos/4", want: "/todos/4"},
		{value: "https://example.test", want: "/todos"},
		{value: "/todos/5", want: "/todos"},
	} {
		if got := editReturnTo(test.value, 4); got != test.want {
			t.Errorf("editReturnTo(%q, 4) = %q, want %q", test.value, got, test.want)
		}
	}
}
