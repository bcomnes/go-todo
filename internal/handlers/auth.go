package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bcomnes/go-todo/internal/middleware"
	"github.com/bcomnes/go-todo/internal/models"
	"github.com/bcomnes/go-todo/pkg/utils"
)

func LoginUser(w http.ResponseWriter, r *http.Request) {
	db, ok := middleware.DBFromContext(r.Context())
	if !ok {
		http.Error(w, "Database connection unavailable", http.StatusInternalServerError)
		return
	}

	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	var user models.User
	err := db.QueryRow(`SELECT id, password_hash FROM users WHERE email = $1`, creds.Email).
		Scan(&user.ID, &user.PasswordHash)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	match, err := utils.CheckPassword(db, user.PasswordHash, creds.Password)
	if err != nil || !match {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := utils.HashPassword(db, time.Now().String()+user.Email)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO auth_tokens (user_id, token, expires_at, created_at, updated_at)
		VALUES ($1, crypt($2, gen_salt('bf')), $3, NOW(), NOW())`, user.ID, token, expiresAt)
	if err != nil {
		http.Error(w, "Failed to store token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
