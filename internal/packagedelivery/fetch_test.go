package packagedelivery

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/packagebundle"
)

func TestHTTPSFetcher(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "http://example.test/plain", http.StatusFound)
			return
		}
		if r.URL.Path == "/bad" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte("verified"))
	}))
	defer server.Close()
	client := server.Client()
	f := HTTPSFetcher{Client: client}
	body, err := f.Open(context.Background(), server.URL)
	must(t, err)
	data, err := io.ReadAll(body)
	body.Close()
	must(t, err)
	if string(data) != "verified" {
		t.Fatal(string(data))
	}
	for _, url := range []string{server.URL + "/redirect", server.URL + "/bad", "http://example.test"} {
		if body, err := f.Open(context.Background(), url); err == nil {
			body.Close()
			t.Fatalf("accepted %s", url)
		}
	}
	if client.CheckRedirect != nil {
		t.Fatal("caller client mutated")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = f.Open(ctx, server.URL)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestFetchRejectsBeforeTreeWrites(t *testing.T) {
	for _, kind := range []string{"outer-pin", "inner-identity", "cap", "bad-archive"} {
		t.Run(kind, func(t *testing.T) {
			home := t.TempDir()
			req, data := fixture(t, home, "1.0.0")
			switch kind {
			case "outer-pin":
				req.SHA256 = "sha256:" + strings.Repeat("0", 64)
			case "inner-identity":
				req.Identity.Version = "2.0.0"
			case "cap":
				data = bytes.Repeat([]byte{0}, int(packagebundle.DefaultLimits.CompressedBytes)+1)
				req.SHA256 = bytesDigest(data)
			case "bad-archive":
				data = []byte("not gzip")
				req.SHA256 = bytesDigest(data)
			}
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data) }))
			defer server.Close()
			req.URL = server.URL
			s := Service{Home: home, Fetcher: HTTPSFetcher{Client: server.Client()}}
			_, err := s.Replace(context.Background(), req, false)
			if err == nil {
				t.Fatal("bad download accepted")
			}
			if _, err := os.Stat(filepath.Join(home, ".patronus", "packages")); !os.IsNotExist(err) {
				t.Fatalf("tree written before verification: %v", err)
			}
		})
	}
}
