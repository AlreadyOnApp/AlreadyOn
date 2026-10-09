//go:build !linux

package governor

import "errors"

// ReadProc is unavailable here; Run holds background analysis.
func ReadProc() (Sample, error) { return Sample{}, errors.New("host CPU is only read on Linux") }

func pauseGroup(int)  {}
func resumeGroup(int) {}
