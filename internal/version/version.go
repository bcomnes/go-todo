//go:generate sh -c "echo 'package version\n\nvar GeneratedCommit = \"'$(git rev-parse HEAD)'\"' > commit_gen.go"

package version

var ServiceName = "go-todo"
var DefaultCommit = "dev"

type Info struct {
    Service string `json:"service"`
    Commit  string `json:"commit"`
}

func Get() Info {
    commit := DefaultCommit
    if GeneratedCommit != "" {
        commit = GeneratedCommit
    }
    return Info{
        Service: ServiceName,
        Commit:  commit,
    }
}
