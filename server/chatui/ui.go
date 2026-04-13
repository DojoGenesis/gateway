// Package chatui embeds the compiled Chat UI SPA.
//
// The dist/ directory is populated by running `make build-chat-spa` (or
// `npm run build` in chat-ui/ then copying to this directory).
// The embedded filesystem is served by the Gateway at GET /chat/*.
//
// If dist/index.html is absent (e.g. on a fresh clone before the SPA is
// built), IsBuilt() returns false and the server serves a 503 with
// instructions instead of panicking at startup.
package chatui

import (
	"embed"
	"io/fs"
)

// dist holds the compiled SPA output. The `all:` prefix includes hidden files
// (like .gitkeep) so the embed compiles even before the SPA is built.
//
//go:embed all:dist
var dist embed.FS

// FS returns the embedded SPA assets rooted at the dist/ directory.
// Use with http.FS or fs.ReadFile to serve individual files.
func FS() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}

// IsBuilt returns true when dist/index.html is present in the embedded
// filesystem, indicating the SPA was compiled before `go build`.
func IsBuilt() bool {
	_, err := dist.Open("dist/index.html")
	return err == nil
}
