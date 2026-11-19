package server

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWithCompressionGzipsLargeJSON(t *testing.T) {
	body := `{"items":[` + strings.Repeat(`{"name":"asset","type":"subdomain"},`, 200) + `{}]}`
	h := withCompression(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/assets", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if !strings.Contains(rec.Header().Get("Vary"), "Accept-Encoding") {
		t.Fatalf("missing Vary: Accept-Encoding, got %q", rec.Header().Get("Vary"))
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != body {
		t.Fatal("decompressed body differs from original")
	}
}

func TestWithCompressionSkipsNonCompressible(t *testing.T) {
	big := strings.Repeat("x", 4096)
	cases := []struct {
		name, contentType, acceptEnc string
	}{
		{"zip download", "application/zip", "gzip"},
		{"binary download", "application/octet-stream", "gzip"},
		{"client without gzip", "application/json", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := withCompression(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", c.contentType)
				_, _ = io.WriteString(w, big)
			}))
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if c.acceptEnc != "" {
				req.Header.Set("Accept-Encoding", c.acceptEnc)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if enc := rec.Header().Get("Content-Encoding"); enc != "" {
				t.Fatalf("Content-Encoding = %q, want none", enc)
			}
			if rec.Body.String() != big {
				t.Fatal("body altered")
			}
		})
	}
}

// An SSE handler that flushes headers before its first event must reach the
// client immediately; the gzip wrapper would hold that flush back.
func TestWithCompressionBypassesEventStreams(t *testing.T) {
	release := make(chan struct{})
	h := withCompression(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush() // headers only, no event yet
		<-release
		fmt.Fprint(w, "id: 1\ndata: {}\n\n")
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()
	defer close(release)

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Accept-Encoding", "gzip")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req) // returns once headers arrive
	if err != nil {
		t.Fatalf("headers were not flushed before the first event: %v", err)
	}
	defer resp.Body.Close()
	if enc := resp.Header.Get("Content-Encoding"); enc != "" {
		t.Fatalf("event stream must not be compressed, got %q", enc)
	}
}
