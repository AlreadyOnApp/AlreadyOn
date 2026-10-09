//go:build !e2e

package main

import "net/http"

// Production and agent previews require TLS, with no plaintext fallback.
func serveListener(server *http.Server) error { return server.ListenAndServeTLS("", "") }
