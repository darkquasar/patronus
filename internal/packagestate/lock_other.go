//go:build !darwin && !linux

package packagestate

// Acquire rejects directory mutation on unsupported operating systems.
func Acquire(_ string) (func() error, error) { return nil, ErrUnsupported }

func sameFilesystem(_, _ string) error { return ErrUnsupported }
