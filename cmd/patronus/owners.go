package main

import "github.com/darkquasar/patronus/internal/state"

// loadOwners loads each ownership state file in order and returns every item.
// pre runs before each load (nil skips it); wrap converts a load error using
// the caller's index and path (nil returns the error as-is). Each caller keeps
// its historical pre-check and wrapping text.
func loadOwners(paths []string, pre func(path string) error, wrap func(i int, path string, err error) error) ([]state.Item, error) {
	var owners []state.Item
	for i, path := range paths {
		if pre != nil {
			if err := pre(path); err != nil {
				return nil, err
			}
		}
		s, err := state.Load(path)
		if err != nil {
			if wrap != nil {
				return nil, wrap(i, path, err)
			}
			return nil, err
		}
		owners = append(owners, s.Items...)
	}
	return owners, nil
}
