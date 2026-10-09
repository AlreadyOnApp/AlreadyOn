//go:build !preview && !e2e

package main

// Normal server builds retain their configured listener addresses.
func listenerAddress(addr string) string { return addr }

// Production HTTP-01 is published separately from encrypted application ports.
func challengeAddress(_ string) string { return ":19902" }
