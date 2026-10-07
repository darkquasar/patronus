package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("catalog-check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	source := flags.String("source", "", "source catalog root")
	index := flags.String("index", "", "local built catalog/index.json")
	closures := flags.String("closures", "", "public profile--target.lock directory")
	profileCases := flags.Bool("profile-cases", false, "emit explicit admitted profile/target metadata as JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *source == "" || *index == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "catalog-check: --source and --index are required; no positional arguments")
		return 2
	}
	c, err := checkCatalog(*source, *index)
	if err == nil {
		err = c.checkLedgers(*closures)
	}
	if err == nil && *profileCases {
		err = c.writeCases(stdout)
	}
	if err != nil {
		fmt.Fprintf(stderr, "catalog-check: %v\n", err)
		return 1
	}
	if !*profileCases {
		fmt.Fprintf(stdout, "catalog packaging valid (%d source manifests); closure coverage %s\n", len(c.items), closureLabel(*closures))
	}
	return 0
}
func closureLabel(dir string) string {
	if dir == "" {
		return "NOT CHECKED (no --closures)"
	}
	return "checked from supplied public locks"
}
