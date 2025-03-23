package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bcomnes/go-todo/internal/version"
)

func Root(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version.Get())
}
