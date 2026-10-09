//go:build !linux

package metrics

type GPUCounter struct{}

func NewGPUCounter() *GPUCounter   { return &GPUCounter{} }
func (*GPUCounter) Read() *float64 { return nil }
func (*GPUCounter) Close()         {}
