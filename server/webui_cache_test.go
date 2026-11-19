package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWebUICacheControl(t *testing.T) {
	cases := []struct {
		name      string
		basicAuth bool
		want      string
	}{
		{"_next/static/chunks/app-3f2a.js", false, "public, max-age=31536000, immutable"},
		{"_next/static/css/x.css", true, "private, max-age=31536000, immutable"},
		{"index.html", false, "no-cache"},
		{"dashboard/index.html", false, "no-cache"},
		{"favicon.ico", false, "no-cache"},
		{"index.html", true, ""}, // keep the Basic Auth gate's no-store
	}
	for _, c := range cases {
		if got := webuiCacheControl(c.name, c.basicAuth); got != c.want {
			t.Errorf("webuiCacheControl(%q, %v) = %q, want %q", c.name, c.basicAuth, got, c.want)
		}
	}
}

func TestWebUIETagsAllowRevalidation(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":                  {Data: []byte("<html>v1</html>")},
		"_next/static/chunks/a.js":    {Data: []byte("console.log(1)")},
		"_next/static/chunks/same.js": {Data: []byte("console.log(1)")},
	}
	tags := webuiETags(fsys)
	if len(tags) != 3 {
		t.Fatalf("got %d tags, want 3: %v", len(tags), tags)
	}
	for name, tag := range tags {
		if !strings.HasPrefix(tag, `W/"`) || !strings.HasSuffix(tag, `"`) {
			t.Fatalf("%s: tag %q is not a weak ETag", name, tag)
		}
	}
	if tags["_next/static/chunks/a.js"] != tags["_next/static/chunks/same.js"] {
		t.Fatal("identical content must yield identical tags")
	}
	if tags["index.html"] == tags["_next/static/chunks/a.js"] {
		t.Fatal("different content must yield different tags")
	}

	// The tag must make http.ServeFileFS answer a conditional GET with 304.
	serve := func(inm string) *httptest.ResponseRecorder {
		// "/" like a real navigation: ServeFileFS redirects paths ending in /index.html.
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if inm != "" {
			req.Header.Set("If-None-Match", inm)
		}
		rec := httptest.NewRecorder()
		rec.Header().Set("ETag", tags["index.html"])
		http.ServeFileFS(rec, req, fsys, "index.html")
		return rec
	}
	if rec := serve(""); rec.Code != http.StatusOK {
		t.Fatalf("plain GET: %d", rec.Code)
	}
	if rec := serve(tags["index.html"]); rec.Code != http.StatusNotModified {
		t.Fatalf("conditional GET with matching tag: %d, want 304", rec.Code)
	}
}
