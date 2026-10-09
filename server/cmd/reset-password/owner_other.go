//go:build !unix

package main

func dropToDataOwner(string) error { return nil }
