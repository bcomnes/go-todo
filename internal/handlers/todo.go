package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bcomnes/go-todo/internal/middleware"
	"github.com/bcomnes/go-todo/internal/models"
	"github.com/bcomnes/go-todo/pkg/utils"
)

func ListTodos(w http.ResponseWriter, r *http.Request) {
	db, ok := middleware.DBFromContext(r.Context())
	user, userOk := middleware.UserFromContext(r.Context())
	if !ok || !userOk {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Parse pagination params
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 10  // Default limit
	offset := 0

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	rows, err := db.Query(`
		SELECT id, task, done, note, created_at, updated_at
		FROM todos
		WHERE user_id = $1
		ORDER BY id
		LIMIT $2 OFFSET $3
	`, user.ID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch todos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var todos []models.Todo
	for rows.Next() {
		var t models.Todo
		if err := rows.Scan(&t.ID, &t.Task, &t.Done, &t.Note, &t.CreatedAt, &t.UpdatedAt); err != nil {
			http.Error(w, "Failed to scan todo", http.StatusInternalServerError)
			return
		}
		todos = append(todos, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todos)
}

func CreateTodo(w http.ResponseWriter, r *http.Request) {
	db, ok := middleware.DBFromContext(r.Context())
	user, userOk := middleware.UserFromContext(r.Context())
	if !ok || !userOk {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var todo models.Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := utils.ValidateStruct(todo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`
		INSERT INTO todos (user_id, task, done, note, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())`,
		user.ID, todo.Task, todo.Done, todo.Note)

	if err != nil {
		http.Error(w, "Failed to create todo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func GetTodo(w http.ResponseWriter, r *http.Request) {
	db, ok := middleware.DBFromContext(r.Context())
	user, userOk := middleware.UserFromContext(r.Context())
	if !ok || !userOk {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid todo ID", http.StatusBadRequest)
		return
	}

	var todo models.Todo
	err = db.QueryRow(`
		SELECT id, task, done, note, created_at, updated_at
		FROM todos
		WHERE id = $1 AND user_id = $2`, id, user.ID).
		Scan(&todo.ID, &todo.Task, &todo.Done, &todo.Note, &todo.CreatedAt, &todo.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Todo not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch todo", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(todo)
}

func UpdateTodo(w http.ResponseWriter, r *http.Request) {
	db, ok := middleware.DBFromContext(r.Context())
	user, userOk := middleware.UserFromContext(r.Context())
	if !ok || !userOk {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid todo ID", http.StatusBadRequest)
		return
	}

	var todo models.Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`
		UPDATE todos SET task = $1, done = $2, note = $3, updated_at = NOW()
		WHERE id = $4 AND user_id = $5`,
		todo.Task, todo.Done, todo.Note, id, user.ID)

	if err != nil {
		http.Error(w, "Failed to update todo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func DeleteTodo(w http.ResponseWriter, r *http.Request) {
	db, ok := middleware.DBFromContext(r.Context())
	user, userOk := middleware.UserFromContext(r.Context())
	if !ok || !userOk {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid todo ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`DELETE FROM todos WHERE id = $1 AND user_id = $2`, id, user.ID)
	if err != nil {
		http.Error(w, "Failed to delete todo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
