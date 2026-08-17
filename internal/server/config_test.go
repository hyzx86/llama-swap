package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testConfigYAML = `
logLevel: info
healthCheckTimeout: 120
models:
  demo:
    cmd: echo ${PORT}
    name: Demo Model
    ttl: 0
peers:
  remote:
    proxy: http://example.com
    models: [remote-model]
`

// newConfigTestServer builds a Server whose /api/config endpoints point at a
// temp config file with the given content.
func newConfigTestServer(t *testing.T, content string) (*Server, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	s := newTestServer(newStubRouter(nil, ""), newStubRouter(nil, ""))
	s.configPath = path
	t.Cleanup(func() { s.store.Close() })
	return s, path
}

func TestServer_GetConfig_ReturnsContent(t *testing.T) {
	s, path := newConfigTestServer(t, testConfigYAML)

	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp configResponse
	require.NoError(t, decodeConfigJSON(t, w.Body, &resp))
	assert.Equal(t, path, resp.Path)
	assert.Equal(t, testConfigYAML, resp.Content)
	assert.Positive(t, resp.MtimeMs)
	assert.Equal(t, int64(len(testConfigYAML)), resp.Size)
}

func TestServer_GetConfig_NoPath_NotImplemented(t *testing.T) {
	s := newTestServer(newStubRouter(nil, ""), newStubRouter(nil, ""))
	t.Cleanup(func() { s.store.Close() })

	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	assert.Equal(t, http.StatusNotImplemented, w.Code)

	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(testConfigYAML)))
	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func TestServer_PutConfig_ValidYAML_WritesFile(t *testing.T) {
	s, path := newConfigTestServer(t, testConfigYAML)

	updated := strings.Replace(testConfigYAML, "healthCheckTimeout: 120", "healthCheckTimeout: 300", 1)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(updated)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, updated, string(data))

	// Permissions must survive the atomic rename.
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	// No temp files may be left behind in the config directory.
	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".llama-swap-config-", "leftover temp file")
	}
}

func TestServer_PutConfig_InvalidYAML_Rejected(t *testing.T) {
	s, path := newConfigTestServer(t, testConfigYAML)

	invalid := "models:\n  demo:\n    cmd: [unclosed\n"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(invalid)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "config validation failed")

	// The file must be untouched.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, testConfigYAML, string(data))
}

func TestServer_PutConfig_UnknownModelKey_StillValid(t *testing.T) {
	// The config loader ignores unknown top-level keys (it decodes into the
	// Config struct), so a key the UI invented is still a valid document.
	s, path := newConfigTestServer(t, testConfigYAML)

	updated := testConfigYAML + "customNote: keep me\n"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(updated)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, updated, string(data))
}

func decodeConfigJSON(t *testing.T, r io.Reader, out any) error {
	t.Helper()
	return json.NewDecoder(r).Decode(out)
}
