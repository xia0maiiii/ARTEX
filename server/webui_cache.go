package server

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"strings"
)

// webuiImmutablePrefix is where Next's static export puts content-hashed
// bundles: a given URL there never changes content.
const webuiImmutablePrefix = "_next/static/"

// webuiETags precomputes an ETag for every file of the embedded UI. embed.FS
// files have a zero ModTime, so net/http emits no Last-Modified and, without an
// ETag, can never answer 304 — every navigation re-downloaded every asset. The
// tags are weak because the compression wrapper may serve a gzip encoding of the
// same representation under the same tag (http.ServeContent compares
// If-None-Match weakly, so revalidation still works).
func webuiETags(fsys fs.FS) map[string]string {
	tags := map[string]string{}
	_ = fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		f, err := fsys.Open(name)
		if err != nil {
			return nil
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return nil
		}
		tags[name] = `W/"` + hex.EncodeToString(h.Sum(nil)[:12]) + `"`
		return nil
	})
	return tags
}

// webuiCacheControl returns the Cache-Control for an embedded UI file, or ""
// to leave the header set by outer middleware untouched.
//
//   - Hashed _next/static assets are immutable for a year. Under Basic Auth they
//     are marked private so shared caches never store gated content (the gate's
//     blanket no-store is overridden only for these hashed bundles).
//   - Everything else (HTML, favicon, …) must revalidate (no-cache + ETag), so
//     an upgrade is picked up immediately. Under Basic Auth the gate's no-store
//     is kept for these.
func webuiCacheControl(name string, basicAuth bool) string {
	if strings.HasPrefix(name, webuiImmutablePrefix) {
		if basicAuth {
			return "private, max-age=31536000, immutable"
		}
		return "public, max-age=31536000, immutable"
	}
	if basicAuth {
		return ""
	}
	return "no-cache"
}
