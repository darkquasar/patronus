package main

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/registry"
)

// deniedFetcher is the FAIL-CLOSED default for every network seam in this
// package's tests. The production defaults (registry.go, install.go) are live HTTP
// clients, so a test that forgets withRemoteEnv would silently reach the real
// internet, pass, and tell nobody. This inverts that: it panics, loudly, with the
// URL it tried to reach.
//
// Never "fix" a panic from here by installing an HTTPFetcher. Serve the bytes from
// memory (see fixtureRegistry / withRemoteEnv). CI must never fetch
// attacker-controllable remote bytes: `go test` runs as the user, with the user's
// credentials, and on CI it runs on every PR — including fork PRs — before human
// review.
type deniedFetcher struct{}

var (
	_ recipe.Fetcher   = deniedFetcher{}
	_ registry.Fetcher = deniedFetcher{}
)

func (deniedFetcher) Fetch(_ context.Context, url string) (io.ReadCloser, error) {
	panic("TEST TRIED TO REACH THE NETWORK: " + url +
		"\nTests must serve bytes from memory (see withRemoteEnv / fixtureRegistry)." +
		"\nNever fetch upstream bytes in a test.")
}

// TestMain denies every fetcher seam by default. A test that wants bytes must
// explicitly install a servingFetcher.
func TestMain(m *testing.M) {
	fetcherForCommands = deniedFetcher{}
	registryFetcher = deniedFetcher{}
	fetcherForDeploy = deniedFetcher{}
	os.Exit(m.Run())
}
