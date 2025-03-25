package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bcomnes/go-todo/internal/config"
	"github.com/bcomnes/go-todo/internal/database"
	"github.com/bcomnes/go-todo/internal/handlers"
	"github.com/bcomnes/go-todo/internal/middleware"
	"github.com/bcomnes/go-todo/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestTodoPagination(t *testing.T) {
	cfg := config.Load("../../.env")
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	var userID int64
	err = db.QueryRow(`INSERT INTO users (username, email, password_hash) VALUES ('test_user_pagination', 'test@example.com', crypt('password', gen_salt('bf'))) RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	// Insert 25 todos
	for i := 0; i < 25; i++ {
		_, err := db.Exec(`INSERT INTO todos (user_id, task, done, note, created_at, updated_at) VALUES ($1, $2, false, NULL, NOW(), NOW())`, userID, fmt.Sprintf("Task %d", i+1))
		if err != nil {
			t.Fatalf("failed to insert todo: %v", err)
		}
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM todos WHERE user_id = $1`, userID)
	})

	// Page 1 test
	page1Req := httptest.NewRequest(http.MethodGet, "/todos?limit=10&offset=0", nil)
	page1Resp := httptest.NewRecorder()
	page1Req = page1Req.WithContext(context.WithValue(page1Req.Context(), middleware.UserContextKey, &models.User{ID: int(userID)}))
	middleware.WithDB(db, handlers.ListTodos)(page1Resp, page1Req)
	if page1Resp.Code != http.StatusOK {
		t.Fatalf("Page 1: expected 200, got %d - body: %s", page1Resp.Code, page1Resp.Body.String())
	}

	var todosPage1 []models.Todo
	err = json.Unmarshal(page1Resp.Body.Bytes(), &todosPage1)
	if err != nil {
		t.Fatalf("Page 1: failed to parse todos: %v", err)
	}
	if len(todosPage1) != 10 {
		t.Fatalf("Page 1: expected 10 todos, got %d", len(todosPage1))
	}

	// Page 3 test (should have 5 left)
	page3Req := httptest.NewRequest(http.MethodGet, "/todos?limit=10&offset=20", nil)
	page3Resp := httptest.NewRecorder()
	page3Req = page3Req.WithContext(context.WithValue(page3Req.Context(), middleware.UserContextKey, &models.User{ID: int(userID)}))
	middleware.WithDB(db, handlers.ListTodos)(page3Resp, page3Req)
	if page3Resp.Code != http.StatusOK {
		t.Fatalf("Page 3: expected 200, got %d - body: %s", page3Resp.Code, page3Resp.Body.String())
	}

	var todosPage3 []models.Todo
	err = json.Unmarshal(page3Resp.Body.Bytes(), &todosPage3)
	if err != nil {
		t.Fatalf("Page 3: failed to parse todos: %v", err)
	}
	if len(todosPage3) != 5 {
		t.Fatalf("Page 3: expected 5 todos, got %d", len(todosPage3))
	}
}
