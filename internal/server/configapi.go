package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mostlygeek/llama-swap/internal/config"
)

// maxConfigSize bounds the request body of PUT /api/config so a stray giant
// upload cannot exhaust disk on the config volume.
const maxConfigSize = 10 << 20 // 10 MiB

// configResponse is the JSON payload of GET /api/config.
type configResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	MtimeMs int64  `json:"mtimeMs"`
	Size    int64  `json:"size"`
}

// handleAPIGetConfig serves the raw config YAML so the UI editor can load it.
// It returns 501 when the server was started without a -config file (e.g.
// -config-dir only), where there is no single file to edit.
func (s *Server) handleAPIGetConfig(w http.ResponseWriter, r *http.Request) {
	if s.configPath == "" {
		sendConfigJSON(w, r, http.StatusNotImplemented, map[string]string{
			"error": "server was started without a -config file; editing is not supported",
		})
		return
	}

	data, err := os.ReadFile(s.configPath)
	if err != nil {
		s.proxylog.Warnf("GET /api/config: failed to read %s: %v", s.configPath, err)
		sendConfigJSON(w, r, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to read config file: %v", err),
		})
		return
	}

	info, err := os.Stat(s.configPath)
	if err != nil {
		sendConfigJSON(w, r, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to stat config file: %v", err),
		})
		return
	}

	sendConfigJSON(w, r, http.StatusOK, configResponse{
		Path:    s.configPath,
		Content: string(data),
		MtimeMs: info.ModTime().UnixMilli(),
		Size:    info.Size(),
	})
}

// handleAPIPutConfig validates the submitted YAML against the config schema
// (via config.LoadConfigFromReader) before writing it back. The write is
// atomic: the new content goes to a temp file in the same directory, is
// synced, keeps the original file's permissions, and is renamed over the
// original. A running --watch-config process picks the change up on its next
// poll and hot-reloads; a failed validation leaves the file untouched.
func (s *Server) handleAPIPutConfig(w http.ResponseWriter, r *http.Request) {
	if s.configPath == "" {
		sendConfigJSON(w, r, http.StatusNotImplemented, map[string]string{
			"error": "server was started without a -config file; editing is not supported",
		})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxConfigSize+1))
	if err != nil {
		sendConfigJSON(w, r, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("failed to read request body: %v", err),
		})
		return
	}
	if int64(len(body)) > maxConfigSize {
		sendConfigJSON(w, r, http.StatusRequestEntityTooLarge, map[string]string{
			"error": fmt.Sprintf("config exceeds maximum size of %d bytes", maxConfigSize),
		})
		return
	}

	// Validate before touching the file: an invalid config must never be
	// written, otherwise the watcher would reload and fail, leaving the
	// running server on a stale config while the file is corrupt.
	if _, err := config.LoadConfigFromReader(bytes.NewReader(body)); err != nil {
		s.proxylog.Warnf("PUT /api/config: rejected invalid config: %v", err)
		sendConfigJSON(w, r, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("config validation failed: %v", err),
		})
		return
	}

	s.configWriteMu.Lock()
	defer s.configWriteMu.Unlock()

	// Preserve the original file's permissions on the replacement.
	mode := os.FileMode(0o644)
	if info, err := os.Stat(s.configPath); err == nil {
		mode = info.Mode().Perm()
	}

	dir := filepath.Dir(s.configPath)
	tmp, err := os.CreateTemp(dir, ".llama-swap-config-*.tmp")
	if err != nil {
		s.proxylog.Warnf("PUT /api/config: failed to create temp file: %v", err)
		sendConfigJSON(w, r, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to create temp file: %v", err),
		})
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename

	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		s.proxylog.Warnf("PUT /api/config: failed to write temp file: %v", err)
		sendConfigJSON(w, r, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to write config file: %v", err),
		})
		return
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		s.proxylog.Warnf("PUT /api/config: failed to sync temp file: %v", err)
		sendConfigJSON(w, r, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to write config file: %v", err),
		})
		return
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		s.proxylog.Warnf("PUT /api/config: failed to set permissions: %v", err)
		sendConfigJSON(w, r, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to set config file permissions: %v", err),
		})
		return
	}
	if err := tmp.Close(); err != nil {
		s.proxylog.Warnf("PUT /api/config: failed to close temp file: %v", err)
		sendConfigJSON(w, r, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to write config file: %v", err),
		})
		return
	}
	if err := os.Rename(tmpName, s.configPath); err != nil {
		s.proxylog.Warnf("PUT /api/config: failed to replace config file: %v", err)
		sendConfigJSON(w, r, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to replace config file: %v", err),
		})
		return
	}

	s.proxylog.Infof("config file updated via API (%d bytes); watcher will hot-reload", len(body))
	sendConfigJSON(w, r, http.StatusOK, map[string]string{"msg": "ok"})
}

// sendConfigJSON writes a JSON body with the given status. The request
// parameter is unused but kept for call-site symmetry with the other
// swaputil send helpers.
func sendConfigJSON(w http.ResponseWriter, _ *http.Request, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// The payload is always small and pre-validated; a marshal failure here
	// would mean the client already got the status code, so just drop it.
	_ = json.NewEncoder(w).Encode(payload)
}
