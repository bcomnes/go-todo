package routes_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bcomnes/go-todo/pkg/httpapi"
)

func TestUnknownRouteIsNotHandledByRoot(t *testing.T) {
	api, err := httpapi.New(&sql.DB{}, time.Hour)
	if err != nil {
		t.Fatalf("initialize API: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
