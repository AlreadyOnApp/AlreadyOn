//go:build !linux && !darwin

package playback

import "errors"

func lockCache(string) (func() error, error) {
	return nil, errors.New("playback: cache locking is supported on Linux/macOS only")
}
