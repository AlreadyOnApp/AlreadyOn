package httpapi

import (
	"bytes"
	"net/http"
	"os"
)

// specMediaServer is the shared application origin written in the spec file.
const specMediaServer = "https://localhost:19900"

// openAPISpecPath is where NewRouter looks for the versioned HTTP API's
// spec (the repo's api/openapi.yaml) to serve at GET /api/openapi.yaml
// and render interactively at GET /api/docs/ (Swagger UI). The default
// is relative to the CWD of a local `make dev` run (the server/ directory,
// a sibling of api/ and web/); the container image overrides it at build
// time with -ldflags -X (see deploy/Dockerfile). It is deliberately not an
// environment variable.
var openAPISpecPath = "../api/openapi.yaml"

// handleOpenAPISpec serves the raw spec file read from disk on every
// request (not cached/embedded), so editing api/openapi.yaml takes
// effect on the next request without a server restart. The application's
// address is replaced by the one the web app itself would use for this request
// (same host name as the docs page), so Swagger UI's Try It Out on media calls
// passes the shared origin check whichever name the docs were opened by.
func handleOpenAPISpec(specPath string, mediaBase func(*http.Request) (string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(specPath)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "OpenAPI spec not found on disk: "+err.Error())
			return
		}
		if mediaBase != nil {
			if base, err := mediaBase(r); err == nil {
				data = bytes.ReplaceAll(data, []byte(specMediaServer), []byte(base))
			}
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(data)
	}
}
