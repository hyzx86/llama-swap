package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_ModelOrder_PreservesYAMLDeclarationOrder(t *testing.T) {
	content := `
models:
  zeta:
    cmd: run zeta
    proxy: http://localhost:9001
  alpha:
    cmd: run alpha
    proxy: http://localhost:9002
  mid:
    cmd: run mid
    proxy: http://localhost:9003
peers:
  peer-b:
    proxy: http://example.com
    models:
      - m2
      - m1
  peer-a:
    proxy: http://example.com
    models:
      - m3
`
	cfg, err := LoadConfigFromReader(strings.NewReader(content))
	require.NoError(t, err)

	assert.Equal(t, []string{"zeta", "alpha", "mid"}, cfg.ModelOrder)
	assert.Equal(t, []string{"peer-b", "peer-a"}, cfg.PeerOrder)

	// The underlying maps are unaffected by the order bookkeeping.
	require.Len(t, cfg.Models, 3)
	require.Len(t, cfg.Peers, 2)
}

func TestConfig_ModelOrder_EmptyWhenSectionsMissing(t *testing.T) {
	cfg, err := LoadConfigFromReader(strings.NewReader(`
logLevel: debug
`))
	require.NoError(t, err)
	assert.Nil(t, cfg.ModelOrder)
	assert.Nil(t, cfg.PeerOrder)
}

func TestConfig_ModelOrder_SurvivesEnvMacroSubstitution(t *testing.T) {
	t.Setenv("LLAMA_SWAP_TEST_PORT", "1234")
	content := `
models:
  first:
    cmd: run first
    proxy: http://localhost:${env.LLAMA_SWAP_TEST_PORT}
  second:
    cmd: run second
    proxy: http://localhost:9002
`
	cfg, err := LoadConfigFromReader(strings.NewReader(content))
	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, cfg.ModelOrder)
}
