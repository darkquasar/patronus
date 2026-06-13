// Package recipe is the engine that makes recipe manifests executable: it fetches
// and verifies external binaries (the §2c self-contained floor), wires them into
// each agent's MCP config (reusing adapter.MergeConfig), and orchestrates
// self-wiring recipes' post-install commands. It produces the same diff.FileDiffs
// the artifact path does, so a recipe install flows through the one change-set
// spine (FETCH + MERGE + EXEC), not a parallel installer.
package recipe

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
)

// Fetcher downloads the bytes at a URL. The httpFetcher is the real impl; tests
// inject a fake so no test touches the network. Verification is deliberately
// NOT part of this interface (see Verify) so the same checksum path runs for
// both real and fake fetchers.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (io.ReadCloser, error)
}

// HTTPFetcher fetches over HTTPS using the stdlib client.
type HTTPFetcher struct {
	Client *http.Client // nil => http.DefaultClient
}

// NewTLS13Fetcher returns an HTTPFetcher whose client requires TLS 1.3 as the
// minimum. It is used ONLY for catalog/registry fetches — an endpoint we control
// (registry.patronus.quasarops.com) — so we can enforce a modern TLS floor there.
// Recipe-binary and git:/https: sourced-reference fetches keep the default client
// (TLS 1.2+), because those point at third-party upstreams whose TLS config we
// don't control and must not break.
func NewTLS13Fetcher() HTTPFetcher {
	return HTTPFetcher{Client: &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13},
		},
	}}
}

// Fetch performs a GET and returns the response body (caller closes).
func (f HTTPFetcher) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	client := f.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("fetch %s: status %s", url, resp.Status)
	}
	return resp.Body, nil
}

// normalizeHex strips an optional "sha256:" prefix and lowercases the digest.
func normalizeHex(s string) string {
	if len(s) > 7 && s[:7] == "sha256:" {
		s = s[7:]
	}
	return toLowerASCII(s)
}

func toLowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
