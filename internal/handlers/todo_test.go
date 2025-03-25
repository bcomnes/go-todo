package handlers_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bcomnes/go-todo/internal/config"
	"github.com/bcomnes/go-todo/internal/database"
	"github.com/bcomnes/go-todo/internal/handlers"
	"github.com/bcomnes/go-todo/internal/middleware"
	"github.com/bcomnes/go-todo/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestTodoCRUD(t *testing.T) {
	cfg := config.Load("../../.env")
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	var userID int64
	err = db.QueryRow(`INSERT INTO users (username, email, password_hash) VALUES ('test_user', 'test@example.com', crypt('password', gen_salt('bf'))) RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	// Create Todo
	todoBody := strings.NewReader(`{"title": "Test Todo", "note": "Test Note"}`)
	req := httptest.NewRequest(http.MethodPost, "/todos", todoBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler := middleware.WithDB(db, handlers.CreateTodo)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, &models.User{ID: int(userID)}))
	handler(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Create: expected 201, got %d - body: %s", w.Code, w.Body.String())
	}

	// Grab inserted todo ID
	var todoID int
	err = db.QueryRow(`SELECT id FROM todos WHERE user_id = $1 ORDER BY id DESC LIMIT 1`, userID).Scan(&todoID)
	if err != nil {
		t.Fatalf("failed to fetch todo ID: %v", err)
	}

	// PATCH update the todo
	patchBody := strings.NewReader(`{"note": "Updated Note"}`)
	patchReq := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/todos/%d", todoID), patchBody)
	patchReq.Header.Set("Content-Type", "application/json")
	patchResp := httptest.NewRecorder()
	patchReq = patchReq.WithContext(context.WithValue(patchReq.Context(), middleware.UserContextKey, &models.User{ID: int(userID)}))
	patchReq.SetPathValue("id", fmt.Sprintf("%d", todoID))
	middleware.WithDB(db, handlers.UpdateTodo)(patchResp, patchReq)
	if patchResp.Code != http.StatusNoContent {
		t.Fatalf("Patch: expected 204 No Content, got %d - body: %s", patchResp.Code, patchResp.Body.String())
	}

	// Verify update
	var updatedNote sql.NullString
	err = db.QueryRow(`SELECT note FROM todos WHERE id = $1`, todoID).Scan(&updatedNote)
	if err != nil || !updatedNote.Valid || updatedNote.String != "Updated Note" {
		t.Fatalf("failed to verify updated note: %v", err)
	}

	// DELETE the todo
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/todos/%d", todoID), nil)
	delResp := httptest.NewRecorder()
	delReq = delReq.WithContext(context.WithValue(delReq.Context(), middleware.UserContextKey, &models.User{ID: int(userID)}))
	delReq.SetPathValue("id", fmt.Sprintf("%d", todoID))
	middleware.WithDB(db, handlers.DeleteTodo)(delResp, delReq)
	if delResp.Code != http.StatusNoContent {
		t.Fatalf("Delete: expected 204 No Content, got %d - body: %s", delResp.Code, delResp.Body.String())
	}

	// Confirm deletion
	var exists bool
	err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM todos WHERE id = $1)`, todoID).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check todo existence: %v", err)
	}
	if exists {
		t.Fatal("todo was not deleted")
	}
}
