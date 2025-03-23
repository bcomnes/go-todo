package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bcomnes/go-todo/internal/middleware"
	"github.com/bcomnes/go-todo/pkg/utils"
)

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	db, ok := middleware.DBFromContext(r.Context())
	if !ok {
		http.Error(w, "Database connection unavailable", http.StatusInternalServerError)
		return
	}

	var req struct {
		Username string `json:"username" validate:"required"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := utils.ValidateStruct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hashedPassword, err := utils.HashPassword(db, req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`
		INSERT INTO users (username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())`,
		req.Username, req.Email, hashedPassword)

	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
