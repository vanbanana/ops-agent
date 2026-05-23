package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FileEntry represents a single file/directory in the listing.
type FileEntry struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "file", "directory", "symlink"
	Size        int64  `json:"size"`
	Mtime       string `json:"mtime"` // ISO 8601
	Permissions string `json:"permissions"`
}

// Blocked paths — never allow access to these
var blockedPaths = []string{
	"/proc", "/sys", "/dev", "/boot",
	"/etc/shadow", "/etc/gshadow", "/etc/sudoers",
}

// Allowed root prefixes — only serve within these
var allowedRoots = []string{
	"/", // Allow browsing from root but block specific dangerous paths
}

// validatePath checks if a path is safe to access.
func validatePath(rawPath string) (string, error) {
	// Clean the path
	cleaned := filepath.Clean(rawPath)
	if !filepath.IsAbs(cleaned) {
		cleaned = "/" + cleaned
	}

	// Block dangerous paths
	for _, blocked := range blockedPaths {
		if cleaned == blocked || strings.HasPrefix(cleaned, blocked+"/") {
			return "", fmt.Errorf("access denied: %s", cleaned)
		}
	}

	// Prevent directory traversal
	if strings.Contains(rawPath, "..") {
		return "", fmt.Errorf("path traversal not allowed")
	}

	return cleaned, nil
}

// HandleFSList handles GET /api/v1/fs/list?path=/some/dir
func HandleFSList(w http.ResponseWriter, r *http.Request) {
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/"
	}

	cleaned, err := validatePath(dirPath)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": 403, "error": err.Error()})
		return
	}

	entries, err := os.ReadDir(cleaned)
	if err != nil {
		status := http.StatusInternalServerError
		if os.IsNotExist(err) {
			status = http.StatusNotFound
		} else if os.IsPermission(err) {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{"code": status, "error": err.Error()})
		return
	}

	result := make([]FileEntry, 0, len(entries))
	for _, entry := range entries {
		fe := FileEntry{Name: entry.Name()}

		info, err := entry.Info()
		if err != nil {
			// Skip entries we can't stat
			continue
		}

		fe.Size = info.Size()
		fe.Mtime = info.ModTime().Format("2006-01-02T15:04:05Z07:00")
		fe.Permissions = info.Mode().Perm().String()

		switch {
		case entry.Type()&fs.ModeSymlink != 0:
			fe.Type = "symlink"
		case entry.IsDir():
			fe.Type = "directory"
		default:
			fe.Type = "file"
		}

		result = append(result, fe)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code":    0,
		"path":    cleaned,
		"entries": result,
	})
}

// HandleFSStat handles GET /api/v1/fs/stat?path=/some/file
func HandleFSStat(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "error": "path required"})
		return
	}

	cleaned, err := validatePath(filePath)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": 403, "error": err.Error()})
		return
	}

	info, err := os.Stat(cleaned)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"code": 404, "error": err.Error()})
		return
	}

	fileType := "file"
	if info.IsDir() {
		fileType = "directory"
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		fileType = "symlink"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"data": map[string]any{
			"name":        info.Name(),
			"type":        fileType,
			"size":        info.Size(),
			"mtime":       info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
			"permissions": info.Mode().Perm().String(),
		},
	})
}

// HandleFSMkdir handles POST /api/v1/fs/mkdir {path: "/some/new/dir"}
func HandleFSMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "error": "invalid body"})
		return
	}

	cleaned, err := validatePath(req.Path)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": 403, "error": err.Error()})
		return
	}

	if err := os.MkdirAll(cleaned, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "message": "created", "path": cleaned})
}

// HandleFSRename handles POST /api/v1/fs/rename {source, destination}
func HandleFSRename(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "error": "invalid body"})
		return
	}

	src, err := validatePath(req.Source)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": 403, "error": err.Error()})
		return
	}
	dst, err := validatePath(req.Destination)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": 403, "error": err.Error()})
		return
	}

	if err := os.Rename(src, dst); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "message": "renamed", "source": src, "destination": dst})
}

// HandleFSCopy handles POST /api/v1/fs/copy {source, destination}
func HandleFSCopy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "error": "invalid body"})
		return
	}

	src, err := validatePath(req.Source)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": 403, "error": err.Error()})
		return
	}
	dst, err := validatePath(req.Destination)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": 403, "error": err.Error()})
		return
	}

	// Simple file copy (not recursive for dirs yet)
	info, err := os.Stat(src)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"code": 404, "error": err.Error()})
		return
	}
	if info.IsDir() {
		// Recursive copy for directories
		if err := copyDir(src, dst); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "error": err.Error()})
			return
		}
	} else {
		if err := copyFile(src, dst); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "error": err.Error()})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "message": "copied", "source": src, "destination": dst})
}

// HandleFSMove handles POST /api/v1/fs/move {source, destination}
func HandleFSMove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "error": "invalid body"})
		return
	}

	src, err := validatePath(req.Source)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": 403, "error": err.Error()})
		return
	}
	dst, err := validatePath(req.Destination)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": 403, "error": err.Error()})
		return
	}

	if err := os.Rename(src, dst); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "message": "moved", "source": src, "destination": dst})
}

// HandleFSDelete handles POST /api/v1/fs/delete {path}
func HandleFSDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "error": "invalid body"})
		return
	}

	cleaned, err := validatePath(req.Path)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": 403, "error": err.Error()})
		return
	}

	// Extra safety: refuse to delete root-level directories
	parts := strings.Split(strings.TrimPrefix(cleaned, "/"), "/")
	if len(parts) <= 1 && cleaned != "/" {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": 403, "error": "cannot delete top-level directory"})
		return
	}

	if err := os.RemoveAll(cleaned); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "message": "deleted", "path": cleaned})
}

// --- Helpers ---

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode().Perm())
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode().Perm())
		}
		return copyFile(path, targetPath)
	})
}
