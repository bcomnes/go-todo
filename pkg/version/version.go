// Package version reports build identity for diagnostics and release tooling.
package version

// Commit is the source revision injected by release builds. Development builds
// retain the default value "dev".
var Commit = "dev"

// Info is the externally reportable service build identity.
type Info struct {
	Service string `json:"service"`
	Commit  string `json:"commit"`
}

// Get returns the current service and build identity.
func Get() Info {
	return Info{
		Service: "go-todo",
		Commit:  Commit,
	}
}
