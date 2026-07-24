// Package web embeds browser assets and provides the generic server-side page
// renderer used by feature route packages. Route packages own their page HTML,
// data model, and allow-listed HTMX fragments; only the document layout is shared.
package web

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"

	"github.com/bcomnes/go-todo/pkg/web/layout"
)

// Page is one independently parsed page template set and its directly renderable
// fragment allow-list.
type Page struct {
	name      string
	template  *template.Template
	fragments map[string]struct{}
}

// NewPage parses source together with the shared document layout. Parsing each
// route page independently prevents template definitions from one route from
// overriding another route's definitions.
func NewPage(name, source string, fragments ...string) (*Page, error) {
	if name == "" {
		return nil, errors.New("web: page name is required")
	}
	parsed, err := template.New(name).Parse(layout.Source())
	if err == nil {
		parsed, err = parsed.Parse(source)
	}
	if err != nil {
		return nil, fmt.Errorf("web: parse %s page: %w", name, err)
	}
	allowed := make(map[string]struct{}, len(fragments))
	for _, fragment := range fragments {
		if fragment == "" {
			return nil, fmt.Errorf("web: %s page has an empty fragment name", name)
		}
		allowed[fragment] = struct{}{}
	}
	return &Page{name: name, template: parsed, fragments: allowed}, nil
}

// RenderPage renders a complete HTML document to w.
func (page *Page) RenderPage(w io.Writer, data any) error {
	return execute(w, page.template, "layout", data)
}

// RenderFragment renders one allow-listed template fragment to w.
func (page *Page) RenderFragment(w io.Writer, fragment string, data any) error {
	if _, ok := page.fragments[fragment]; !ok {
		return fmt.Errorf("web: unknown fragment %q for page %q", fragment, page.name)
	}
	return execute(w, page.template, fragment, data)
}

func execute(w io.Writer, tmpl *template.Template, name string, data any) error {
	var buffer bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buffer, name, data); err != nil {
		return fmt.Errorf("web: render %s: %w", name, err)
	}
	if _, err := io.Copy(w, &buffer); err != nil {
		return fmt.Errorf("web: write %s: %w", name, err)
	}
	return nil
}
