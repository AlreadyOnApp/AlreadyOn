package httpapi

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type browsedFolder struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type folderBrowserResponse struct {
	Path    string          `json:"path"`
	Parent  string          `json:"parent"`
	Folders []browsedFolder `json:"folders"`
}

// Mounted under the authenticated management API. Lists directories only;
// access uses the same OS permissions as the library scanner.
func handleBrowseFolders(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/data/media/"
	}
	if !filepath.IsAbs(path) || strings.ContainsRune(path, 0) {
		writeJSONError(w, http.StatusBadRequest, "Choose an absolute folder path.")
		return
	}
	path = filepath.Clean(path)
	entries, err := os.ReadDir(path)
	if err != nil {
		status, message := http.StatusBadRequest, "This path cannot be opened as a folder."
		if errors.Is(err, os.ErrNotExist) {
			status, message = http.StatusNotFound, "This folder does not exist. Navigate up or enter another path."
		}
		if errors.Is(err, os.ErrPermission) {
			status, message = http.StatusForbidden, "The server does not have permission to read this folder."
		}
		writeJSONError(w, status, message)
		return
	}
	result := folderBrowserResponse{Path: path, Parent: filepath.Dir(path), Folders: []browsedFolder{}}
	if result.Parent == path {
		result.Parent = ""
	}
	for _, entry := range entries {
		directory := entry.IsDir()
		if entry.Type()&os.ModeSymlink != 0 {
			info, err := os.Stat(filepath.Join(path, entry.Name()))
			directory = err == nil && info.IsDir()
		}
		if directory {
			result.Folders = append(result.Folders, browsedFolder{Name: entry.Name(), Path: filepath.Join(path, entry.Name())})
		}
	}
	sort.Slice(result.Folders, func(i, j int) bool {
		a, b := result.Folders[i].Name, result.Folders[j].Name
		if strings.ToLower(a) == strings.ToLower(b) {
			return a < b
		}
		return strings.ToLower(a) < strings.ToLower(b)
	})
	writeJSON(w, http.StatusOK, result)
}
