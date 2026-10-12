package portal_test

import (
	"os"
	"path/filepath"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMapStore(t *testing.T) (*portal.MapStore, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "rfid_map.json")
	return portal.NewMapStore(path, filepath.Join(dir, "map.lock")), path
}

func TestMapStoreLoadMissingIsEmpty(t *testing.T) {
	m, _ := newTestMapStore(t)
	data, err := m.Load()
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestMapStoreAssignAndRemove(t *testing.T) {
	m, _ := newTestMapStore(t)
	require.NoError(t, m.Assign("123", "/music/a"))
	require.NoError(t, m.Assign("456", "/music/b"))

	data, err := m.Load()
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"123": "/music/a", "456": "/music/b"}, data)

	require.NoError(t, m.Remove("123"))
	data, err = m.Load()
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"456": "/music/b"}, data)
}

func TestMapStoreWritesIndentedJSON(t *testing.T) {
	m, path := newTestMapStore(t)
	require.NoError(t, m.Assign("123", "/music/a"))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "    \"123\": \"/music/a\"")
}

func TestMapStoreUpdatePreservesExistingKeys(t *testing.T) {
	m, _ := newTestMapStore(t)
	require.NoError(t, m.Assign("1", "/a"))
	require.NoError(t, m.Update(func(d map[string]string) error {
		d["2"] = "/b"
		return nil
	}))
	data, err := m.Load()
	require.NoError(t, err)
	assert.Len(t, data, 2)
}
