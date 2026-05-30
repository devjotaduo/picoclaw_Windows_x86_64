package server

import (
	"embed"
	"io/fs"
)

// distFS holds the built Vite frontend. The dist/ directory always contains at
// least a placeholder index.html so the package compiles before a UI build.
//
//go:embed all:dist
var distFS embed.FS

// uiFS returns the embedded UI rooted at dist/.
func uiFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err) // dist/ is embedded above, so this cannot fail
	}
	return sub
}
