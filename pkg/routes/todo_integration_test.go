package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bcomnes/go-todo/pkg/database"
	"github.com/bcomnes/go-todo/pkg/httpapi"
	"github.com/bcomnes/go-todo/pkg/models"
)

type todoIntegrationLoginResponse struct {
	Token string `json:"token"`
}

func TestTodoLifecycle(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	db, err := database.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	api, err := httpapi.New(db, time.Hour)
	if err != nil {
		t.Fatalf("initialize API: %v", err)
	}
	handler := api.Handler()

	suffix := time.Now().UnixNano()
	firstUsername := fmt.Sprintf("todouser%d", suffix)
	firstEmail := fmt.Sprintf("todouser%d@example.test", suffix)
	secondUsername := fmt.Sprintf("todoother%d", suffix)
	secondEmail := fmt.Sprintf("todoother%d@example.test", suffix)
	password := "correct horse battery staple"
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM public.users WHERE email IN ($1, $2)`, firstEmail, secondEmail)
	})

	firstToken := todoIntegrationRegisterAndLogin(t, handler, firstUsername, firstEmail, password)
	secondToken := todoIntegrationRegisterAndLogin(t, handler, secondUsername, secondEmail, password)

	initialNote := "Created through the JSON API"
	createResponse := todoIntegrationJSONRequest(t, handler, http.MethodPost, "/api/todos", firstToken, map[string]any{
		"task": "Test the todo API",
		"done": false,
		"note": initialNote,
	})
	todoIntegrationRequireStatus(t, createResponse, http.StatusCreated, "create todo")
	todoIntegrationAssertNoUserID(t, createResponse.Body.Bytes())
	var created models.Todo
	todoIntegrationDecodeJSON(t, createResponse, &created, "create todo")
	if created.ID <= 0 {
		t.Fatalf("created todo ID = %d, want a positive ID", created.ID)
	}
	if created.Task != "Test the todo API" || created.Done || created.Note == nil || *created.Note != initialNote {
		t.Fatalf("created todo = %#v", created)
	}

	todoPath := fmt.Sprintf("/api/todos/%d", created.ID)
	getResponse := todoIntegrationJSONRequest(t, handler, http.MethodGet, todoPath, firstToken, nil)
	todoIntegrationRequireStatus(t, getResponse, http.StatusOK, "get todo")
	todoIntegrationAssertNoUserID(t, getResponse.Body.Bytes())
	var fetched models.Todo
	todoIntegrationDecodeJSON(t, getResponse, &fetched, "get todo")
	if fetched.ID != created.ID || fetched.Task != created.Task {
		t.Fatalf("fetched todo = %#v, want ID %d and task %q", fetched, created.ID, created.Task)
	}

	listResponse := todoIntegrationJSONRequest(t, handler, http.MethodGet, "/api/todos", firstToken, nil)
	todoIntegrationRequireStatus(t, listResponse, http.StatusOK, "list todos")
	todoIntegrationAssertNoUserID(t, listResponse.Body.Bytes())
	var listed []models.Todo
	todoIntegrationDecodeJSON(t, listResponse, &listed, "list todos")
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("listed todos = %#v, want only todo %d", listed, created.ID)
	}

	otherUserGetResponse := todoIntegrationJSONRequest(t, handler, http.MethodGet, todoPath, secondToken, nil)
	todoIntegrationRequireStatus(t, otherUserGetResponse, http.StatusNotFound, "get another user's todo")

	patchResponse := todoIntegrationJSONRequest(t, handler, http.MethodPatch, todoPath, firstToken, json.RawMessage(`{
		"task": "Updated through the JSON API",
		"done": true,
		"note": null
	}`))
	todoIntegrationRequireStatus(t, patchResponse, http.StatusOK, "update todo")
	todoIntegrationAssertNoUserID(t, patchResponse.Body.Bytes())
	var updated models.Todo
	todoIntegrationDecodeJSON(t, patchResponse, &updated, "update todo")
	if updated.Task != "Updated through the JSON API" || !updated.Done {
		t.Fatalf("updated todo = %#v", updated)
	}
	if updated.Note != nil {
		t.Fatalf("updated todo note = %q, want explicit JSON null to clear it", *updated.Note)
	}

	sessionCookie := todoIntegrationBrowserLogin(t, handler, firstEmail, password)
	todosPageRequest := httptest.NewRequest(http.MethodGet, "/todos", nil)
	todosPageRequest.AddCookie(sessionCookie)
	todosPageResponse := httptest.NewRecorder()
	handler.ServeHTTP(todosPageResponse, todosPageRequest)
	todoIntegrationRequireStatus(t, todosPageResponse, http.StatusOK, "get todos page")
	if !strings.Contains(todosPageResponse.Body.String(), "Updated through the JSON API") {
		t.Fatal("todos page does not contain the authenticated user's todo")
	}

	htmxTask := fmt.Sprintf("HTMX todo %d", suffix)
	htmxForm := url.Values{
		"task": {htmxTask},
		"note": {"Created through an HTMX mutation"},
	}
	htmxRequest := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(htmxForm.Encode()))
	htmxRequest.AddCookie(sessionCookie)
	htmxRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	htmxRequest.Header.Set("Origin", "https://example.com")
	htmxRequest.Header.Set("HX-Request", "true")
	htmxResponse := httptest.NewRecorder()
	handler.ServeHTTP(htmxResponse, htmxRequest)
	todoIntegrationRequireStatus(t, htmxResponse, http.StatusOK, "create todo through HTMX")
	fragment := strings.TrimSpace(htmxResponse.Body.String())
	if !strings.HasPrefix(fragment, `<section id="todo-list"`) {
		t.Fatalf("HTMX response is not the todo-list fragment: %s", fragment)
	}
	if strings.Contains(strings.ToLower(fragment), "<!doctype html>") || strings.Contains(fragment, `id="todo-create-form"`) {
		t.Fatal("HTMX todo mutation unexpectedly returned the full page")
	}
	if !strings.Contains(fragment, htmxTask) {
		t.Fatalf("HTMX todo fragment does not contain mutation %q", htmxTask)
	}

	invalidForm := url.Values{"task": {"   "}}
	invalidRequest := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(invalidForm.Encode()))
	invalidRequest.AddCookie(sessionCookie)
	invalidRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidRequest.Header.Set("Origin", "https://example.com")
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalidRequest)
	todoIntegrationRequireStatus(t, invalidResponse, http.StatusBadRequest, "submit invalid todo without HTMX")
	if !strings.Contains(strings.ToLower(invalidResponse.Body.String()), "<!doctype html>") ||
		!strings.Contains(invalidResponse.Body.String(), `role="alert"`) {
		t.Fatal("ordinary invalid form submission did not render a full error page")
	}

	deleteResponse := todoIntegrationJSONRequest(t, handler, http.MethodDelete, todoPath, firstToken, nil)
	todoIntegrationRequireStatus(t, deleteResponse, http.StatusNoContent, "delete todo")

	getDeletedResponse := todoIntegrationJSONRequest(t, handler, http.MethodGet, todoPath, firstToken, nil)
	todoIntegrationRequireStatus(t, getDeletedResponse, http.StatusNotFound, "get deleted todo")
}

func todoIntegrationRegisterAndLogin(t *testing.T, handler http.Handler, username, email, password string) string {
	t.Helper()

	registerResponse := todoIntegrationJSONRequest(t, handler, http.MethodPost, "/api/register", "", map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	})
	todoIntegrationRequireStatus(t, registerResponse, http.StatusCreated, "register user")

	loginResponse := todoIntegrationJSONRequest(t, handler, http.MethodPost, "/api/login", "", map[string]string{
		"email":    email,
		"password": password,
	})
	todoIntegrationRequireStatus(t, loginResponse, http.StatusOK, "log in user")
	var login todoIntegrationLoginResponse
	todoIntegrationDecodeJSON(t, loginResponse, &login, "log in user")
	if login.Token == "" {
		t.Fatal("login returned an empty token")
	}
	return login.Token
}

func todoIntegrationBrowserLogin(t *testing.T, handler http.Handler, email, password string) *http.Cookie {
	t.Helper()

	form := url.Values{"email": {email}, "password": {password}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://example.com")
	request.Header.Set("HX-Request", "true")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	todoIntegrationRequireStatus(t, response, http.StatusNoContent, "browser login")

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("browser login cookie count = %d, want 1", len(cookies))
	}
	return cookies[0]
}

func todoIntegrationJSONRequest(t *testing.T, handler http.Handler, method, target, token string, payload any) *httptest.ResponseRecorder {
	t.Helper()

	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode %s %s request: %v", method, target, err)
		}
		body = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, target, body)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func todoIntegrationRequireStatus(t *testing.T, response *httptest.ResponseRecorder, want int, operation string) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("%s status = %d, want %d, body = %s", operation, response.Code, want, response.Body.String())
	}
}

func todoIntegrationDecodeJSON(t *testing.T, response *httptest.ResponseRecorder, destination any, operation string) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), destination); err != nil {
		t.Fatalf("decode %s response: %v; body = %s", operation, err, response.Body.String())
	}
}

func todoIntegrationAssertNoUserID(t *testing.T, body []byte) {
	t.Helper()
	if bytes.Contains(body, []byte(`"user_id"`)) {
		t.Fatalf("todo response exposes user_id: %s", body)
	}
}
