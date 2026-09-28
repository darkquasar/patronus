package packagedelivery

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/darkquasar/patronus/internal/packagebundle"
)

// HTTPSFetcher rejects insecure initial URLs and redirects without modifying Client.
type HTTPSFetcher struct{ Client *http.Client }

var _ Fetcher = HTTPSFetcher{}

// Open returns a successful HTTPS response body, owned by the caller.
func (f HTTPSFetcher) Open(ctx context.Context, address string) (io.ReadCloser, error) {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return nil, errors.New("package fetch requires HTTPS without credentials")
	}
	client := http.Client{Timeout: 5 * time.Minute}
	if f.Client != nil {
		client = *f.Client
	}
	redirect := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" || req.URL.User != nil {
			return errors.New("insecure package redirect")
		}
		if len(via) >= 10 {
			return errors.New("too many package redirects")
		}
		if redirect != nil {
			return redirect(req, via)
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("package fetch HTTP %d", response.StatusCode)
	}
	return response.Body, nil
}
func (s *Service) fetch(ctx context.Context, req Request) (*packagebundle.Bundle, error) {
	fetcher := s.Fetcher
	if fetcher == nil {
		fetcher = HTTPSFetcher{}
	}
	body, err := fetcher.Open(ctx, req.URL)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	temp, err := os.CreateTemp("", "patronus-package-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()
	hash := sha256.New()
	limited := io.LimitReader(body, packagebundle.DefaultLimits.CompressedBytes+1)
	n, err := io.Copy(io.MultiWriter(temp, hash), contextReader{ctx: ctx, reader: limited})
	if err != nil {
		return nil, err
	}
	if n > packagebundle.DefaultLimits.CompressedBytes {
		return nil, errors.New("compressed package exceeds limit")
	}
	if fmt.Sprintf("sha256:%x", hash.Sum(nil)) != req.SHA256 {
		return nil, errors.New("package archive checksum mismatch")
	}
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return packagebundle.Decode(contextReader{ctx: ctx, reader: temp}, req.Identity, packagebundle.DefaultLimits)
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
