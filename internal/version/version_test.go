package version_test

import (
	"testing"

	"github.com/bcomnes/go-todo/internal/version"
)

func TestGetVersionInfo(t *testing.T) {
	info := version.Get()

	if info.Service != "go-todo" {
		t.Errorf("expected service 'go-todo', got '%s'", info.Service)
	}

	if info.Commit == "" {
		t.Error("expected non-empty commit hash")
	}
}
