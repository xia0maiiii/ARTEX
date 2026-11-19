//go:build embedui

package server

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// webuiDist holds the statically-exported frontend (built with
// `cd web && npm run build:static`, then copied into server/webui/dist). Compiled
// into the binary only when building with `-tags embedui`. The `all:` prefix is
// required so Next's `_next/` asset dir (leading underscore) is included.
//
//go:embed all:webui/dist
var webuiDist embed.FS

// webuiHandler serves the embedded SPA. Public (no JWT) — auth is enforced
// client-side and on /api. Serves the exported per-route index.html files and
// falls back to index.html so client-side routing still resolves unknown paths.
func (s *Server) webuiHandler() http.Handler {
	root, err := fs.Sub(webuiDist, "webui/dist")
	if err != nil {
		panic(err)
	}
	etags := webuiETags(root)
	serve := func(w http.ResponseWriter, r *http.Request, name string) {
		h := w.Header()
		if tag := etags[name]; tag != "" {
			h.Set("ETag", tag) // ServeFileFS answers If-None-Match with 304
		}
		if cc := webuiCacheControl(name, s.basicAuth.snapshot().Enabled); cc != "" {
			h.Set("Cache-Control", cc)
		}
		http.ServeFileFS(w, r, root, name)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		// 1) exact file (assets: _next/*, favicon.ico, ...)
		// 2) route dir → <p>/index.html (trailingSlash export)
		// 3) <p>.html
		// 4) SPA 兜底 → index.html（交给客户端路由）
		for _, cand := range []string{p, p + "/index.html", p + ".html"} {
			if isRegularFile(root, cand) {
				serve(w, r, cand)
				return
			}
		}
		serve(w, r, "index.html")
	})
}

// isRegularFile reports whether name exists in fsys as a regular file.
func isRegularFile(fsys fs.FS, name string) bool {
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	st, statErr := f.Stat()
	_ = f.Close()
	return statErr == nil && !st.IsDir()
}
