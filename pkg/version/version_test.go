package version_test

import (
	"testing"

	"github.com/bcomnes/go-todo/pkg/version"
)

func TestGetVersionInfo(t *testing.T) {
	info := version.Get()
	if info.Service != "go-todo" {
		t.Errorf("service = %q, want go-todo", info.Service)
	}
	if info.Commit == "" {
		t.Error("commit is empty")
	}
}
