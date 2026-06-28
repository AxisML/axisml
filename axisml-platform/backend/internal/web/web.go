// Package web carries the built Platform SPA so the backend can serve the
// frontend itself — a single image, single binary, same-origin (no CORS, no
// separate gateway route).
//
// The bundle under dist/ is produced by `pnpm run build` in the frontend
// component and staged here at build time (the Dockerfile's frontend stage, or
// `make frontend-stage` for local builds). A committed placeholder index.html
// keeps `go build` working when the real bundle hasn't been staged; dev loops
// use the Vite dev server (which proxies /api to :8080) instead.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var bundle embed.FS

// FS returns the built SPA bundle rooted at dist/.
func FS() (fs.FS, error) {
	return fs.Sub(bundle, "dist")
}
