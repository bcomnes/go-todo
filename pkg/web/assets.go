package web

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var embeddedFiles embed.FS

var assets = mustSub(embeddedFiles, "dist")

// Assets returns the esbuild output rooted at the embedded dist directory. Asset
// compilation is an external build step; the Go server only serves these files.
func Assets() fs.FS {
	return assets
}

func mustSub(files fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(files, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
