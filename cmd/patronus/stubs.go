package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// stub describes a command planned for a later phase.
type stub struct {
	use   string
	short string
	phase string
}

var stubs = []stub{
	{"remove", "Cleanly uninstall tracked items", "8"},
	{"init", "Scaffold a fresh project/global config", "3"},
}

func addStubCommands(root *cobra.Command) {
	for _, s := range stubs {
		s := s
		root.AddCommand(&cobra.Command{
			Use:                s.use,
			Short:              s.short + " (not yet implemented)",
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Fprintf(cmd.ErrOrStderr(),
					"patronus %s: not yet implemented (planned for Phase %s)\n", s.use, s.phase)
				return nil
			},
		})
	}
}
