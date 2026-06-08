// Command patronus is the meta-scaffolder CLI for AI coding environments.
package main

import "os"

// version is the build version, injected at release time via
// -ldflags "-X main.version=<tag>". It defaults to "dev" for local builds and is
// surfaced by `patronus version` / `--version`.
var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
