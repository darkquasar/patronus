// Command patronus is the meta-scaffolder CLI for AI coding environments.
package main

import "os"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
