//go:build !linux

package metrics

func capacity(string) (int64, int64, uint64, bool) { return 0, 0, 0, false }
