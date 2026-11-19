package server

import (
	"net/http"
	"strings"

	"github.com/klauspost/compress/gzhttp"
)

// compressibleTypes is an allowlist: only text formats worth compressing.
// Archives, images and octet-stream downloads pass through untouched.
var compressibleTypes = []string{
	"text/html",
	"text/css",
	"text/plain",
	"text/javascript",
	"application/javascript",
	"application/json",
	"application/manifest+json",
	"image/svg+xml",
}

// withCompression gzips compressible responses for clients that accept it.
//
// Event streams bypass the wrapper entirely. gzhttp buffers writes until it has
// enough bytes to decide whether to compress, and ignores a Flush issued before
// any body write — so an SSE handler's initial header flush would be held back
// until the first event or keep-alive ping. Every SSE consumer (browser
// EventSource, MCP streamable-HTTP clients) advertises text/event-stream in
// Accept, which is what the bypass keys on.
func withCompression(next http.Handler) http.Handler {
	wrap, err := gzhttp.NewWrapper(
		gzhttp.MinSize(1024),
		gzhttp.ContentTypes(compressibleTypes),
	)
	if err != nil {
		panic(err) // static options; only fails on programmer error
	}
	gz := wrap(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			next.ServeHTTP(w, r)
			return
		}
		gz.ServeHTTP(w, r)
	})
}
