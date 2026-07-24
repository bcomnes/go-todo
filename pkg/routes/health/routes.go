// Package health owns the process liveness endpoint.
package health

import (
	"net/http"

	"github.com/bcomnes/go-todo/pkg/httpx"
)

// Register adds the process liveness endpoint to mux.
func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", get)
}

func get(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
