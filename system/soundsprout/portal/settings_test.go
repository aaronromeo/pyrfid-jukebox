package portal_test

import (
	"os"
	"path/filepath"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSettings(t *testing.T) (*portal.Settings, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "conf")
	return portal.NewSettings(path, filepath.Join(dir, "settings.lock")), path
}

func TestSettingsGetMissing(t *testing.T) {
	s, _ := newTestSettings(t)
	v, err := s.Get("PJ_BLUETOOTH_DEVICE")
	require.NoError(t, err)
	assert.Equal(t, "", v)
}

func TestSettingsSetAndGet(t *testing.T) {
	s, _ := newTestSettings(t)
	require.NoError(t, s.Set("PJ_BLUETOOTH_DEVICE", "AA:BB:CC:DD:EE:FF"))
	v, err := s.Get("PJ_BLUETOOTH_DEVICE")
	require.NoError(t, err)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", v)
}

func TestSettingsSetPreservesOtherKeys(t *testing.T) {
	s, path := newTestSettings(t)
	require.NoError(t, os.WriteFile(path, []byte("PJ_DEFAULT_BRANCH=main\n"), 0o644))
	require.NoError(t, s.Set("PJ_BLUETOOTH_DEVICE", "AA:BB:CC:DD:EE:FF"))
	branch, err := s.Get("PJ_DEFAULT_BRANCH")
	require.NoError(t, err)
	assert.Equal(t, "main", branch)
	device, err := s.Get("PJ_BLUETOOTH_DEVICE")
	require.NoError(t, err)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", device)
}

func TestSettingsSetReplacesExistingLine(t *testing.T) {
	s, path := newTestSettings(t)
	require.NoError(t, os.WriteFile(path, []byte("PJ_BLUETOOTH_DEVICE=old\n"), 0o644))
	require.NoError(t, s.Set("PJ_BLUETOOTH_DEVICE", "new"))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "PJ_BLUETOOTH_DEVICE=new\n", string(raw))
}
