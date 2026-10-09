//go:build !linux

package governor

import (
	"errors"
	"time"
)

// Background analysis is held when CPU accounting is unavailable.
func processCPU(int) (time.Duration, error) {
	return 0, errors.New("analysis CPU budget requires Linux")
}
