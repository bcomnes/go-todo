package web

import (
	"bytes"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/bcomnes/go-todo/pkg/models"
	"github.com/bcomnes/go-todo/pkg/web/layout"
)

const testPageSource = `
{{define "content" -}}
<section id="page"><h1>{{.Message}}</h1>{{template "card" .}}</section>
{{- end}}
{{define "card" -}}
<article id="card">{{if .Error}}<p role="alert">{{.Error}}</p>{{end}}</article>
{{- end}}
`

type testPageData struct {
	layout.Data
	Message string
	Error   string
}

func TestRenderPage(t *testing.T) {
	t.Parallel()
	page, err := NewPage("test", testPageSource, "card")
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	var output bytes.Buffer
	if err := page.RenderPage(&output, testPageData{Data: layout.Data{Title: "Test"}, Message: "Hello"}); err != nil {
		t.Fatalf("RenderPage: %v", err)
	}
	for _, want := range []string{
		"<!doctype html>",
		`href="/assets/global.css"`,
		`src="/assets/global.js"`,
		`<main id="main-content">`,
		`id="card"`,
		"Hello",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("rendered page does not contain %q", want)
		}
	}
}

func TestNavigationReflectsAuthentication(t *testing.T) {
	t.Parallel()
	page, err := NewPage("test", testPageSource, "card")
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	var anonymous bytes.Buffer
	if err := page.RenderPage(&anonymous, testPageData{}); err != nil {
		t.Fatalf("RenderPage anonymous: %v", err)
	}
	if !strings.Contains(anonymous.String(), `href="/login"`) || !strings.Contains(anonymous.String(), `href="/register"`) {
		t.Error("anonymous navigation does not contain login and registration links")
	}
	if strings.Contains(anonymous.String(), `hx-post="/logout"`) {
		t.Error("anonymous navigation contains logout form")
	}

	var authenticated bytes.Buffer
	data := testPageData{Data: layout.Data{CurrentUser: &models.User{Username: "ada", Email: "ada@example.com"}}}
	if err := page.RenderPage(&authenticated, data); err != nil {
		t.Fatalf("RenderPage authenticated: %v", err)
	}
	if !strings.Contains(authenticated.String(), `href="/account"`) || !strings.Contains(authenticated.String(), `hx-post="/logout"`) {
		t.Error("authenticated navigation does not contain account link and logout form")
	}
	if !strings.Contains(authenticated.String(), "Log out ada") {
		t.Error("authenticated navigation does not identify the current user")
	}
}

func TestRenderFragment(t *testing.T) {
	t.Parallel()
	page, err := NewPage("test", testPageSource, "card")
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	var output bytes.Buffer
	if err := page.RenderFragment(&output, "card", testPageData{Error: "Try again"}); err != nil {
		t.Fatalf("RenderFragment: %v", err)
	}
	if !strings.Contains(output.String(), `id="card"`) || !strings.Contains(output.String(), "Try again") {
		t.Errorf("rendered fragment is incomplete: %s", output.String())
	}
	if strings.Contains(output.String(), "<!doctype html>") || strings.Contains(output.String(), "/assets/global.css") {
		t.Error("fragment unexpectedly contains the shared layout")
	}
	if err := page.RenderFragment(&output, "other", testPageData{}); err == nil {
		t.Fatal("unknown fragment error = nil")
	}
}

func TestRenderEscapesData(t *testing.T) {
	t.Parallel()
	page, err := NewPage("test", testPageSource, "card")
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	var output bytes.Buffer
	if err := page.RenderFragment(&output, "card", testPageData{Error: `<script>alert("unsafe")</script>`}); err != nil {
		t.Fatalf("RenderFragment: %v", err)
	}
	if strings.Contains(output.String(), "<script>") || !strings.Contains(output.String(), "&lt;script&gt;") {
		t.Errorf("rendered fragment did not safely escape data: %s", output.String())
	}
}

func TestRenderDoesNotWritePartialOutput(t *testing.T) {
	t.Parallel()
	page, err := NewPage("test", testPageSource, "card")
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	output := bytes.NewBufferString("existing")
	data := struct {
		Title       string
		CurrentUser failingUser
		Message     string
		Error       string
	}{CurrentUser: failingUser{}}
	if err := page.RenderPage(output, data); err == nil {
		t.Fatal("RenderPage error = nil, want template execution error")
	}
	if got := output.String(); got != "existing" {
		t.Errorf("output = %q, want unchanged buffer", got)
	}
}

func TestNewPageRejectsInvalidTemplates(t *testing.T) {
	t.Parallel()
	if _, err := NewPage("", testPageSource); err == nil {
		t.Fatal("empty page name error = nil")
	}
	if _, err := NewPage("broken", `{{define "content"}}`); err == nil {
		t.Fatal("invalid template error = nil")
	}
	if _, err := NewPage("empty-fragment", testPageSource, ""); err == nil {
		t.Fatal("empty fragment error = nil")
	}
}

func TestAssets(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"global.css", "global.js"} {
		asset, err := fs.ReadFile(Assets(), name)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", name, err)
		}
		if len(asset) == 0 {
			t.Errorf("ReadFile(%q) returned an empty asset", name)
		}
	}
}

type failingUser struct{}

func (failingUser) Username() (string, error) {
	return "", errors.New("username unavailable")
}
