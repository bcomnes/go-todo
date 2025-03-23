package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/bcomnes/go-todo/internal/models"
)

const UserContextKey contextKey = "user"

func WithAuth(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		var user models.User
		err := db.QueryRow(`
			SELECT users.id, users.username, users.email, users.password_hash, users.created_at, users.updated_at
			FROM auth_tokens
			JOIN users ON auth_tokens.user_id = users.id
			WHERE auth_tokens.token = crypt($1, auth_tokens.token) AND auth_tokens.expires_at > NOW()
		`, token).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)

		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, &user)
		next(w, r.WithContext(ctx))
	}
}

func UserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	return user, ok
}
