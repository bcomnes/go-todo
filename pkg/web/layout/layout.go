// Package layout embeds the document shell shared by all server-rendered pages.
package layout

import (
	_ "embed"

	"github.com/bcomnes/go-todo/pkg/models"
)

//go:embed layout.gohtml
var source string

// Data contains values used by the shared document layout.
type Data struct {
	Title       string
	CurrentUser *models.User
}

// Source returns the shared document layout template.
func Source() string {
	return source
}
