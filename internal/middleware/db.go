package middleware

import (
	"context"
	"database/sql"
	"net/http"
)

type contextKey string

const dbContextKey contextKey = "db"

func WithDB(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), dbContextKey, db)
		next(w, r.WithContext(ctx))
	}
}

func DBFromContext(ctx context.Context) (*sql.DB, bool) {
	db, ok := ctx.Value(dbContextKey).(*sql.DB)
	return db, ok
}
