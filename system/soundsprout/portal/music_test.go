package portal_test

import (
	"os"
	"path/filepath"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLibraryFoldersCountsMP3(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "card-1"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "card-1", "a.mp3"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "card-1", "b.txt"), []byte("x"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "card-2"), 0o755))

	folders, err := portal.NewLibrary(root).Folders()
	require.NoError(t, err)
	require.Len(t, folders, 2)
	assert.Equal(t, "card-1", folders[0].Name)
	assert.Equal(t, 1, folders[0].MP3Count)
	assert.Equal(t, 0, folders[1].MP3Count)
}

func TestLibraryResolveRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	lib := portal.NewLibrary(root)
	_, err := lib.Resolve("../etc")
	require.Error(t, err)
	_, err = lib.Resolve("a/b")
	require.Error(t, err)
}

func TestLibraryResolveRequiresExistingDir(t *testing.T) {
	root := t.TempDir()
	lib := portal.NewLibrary(root)
	_, err := lib.Resolve("nope")
	require.Error(t, err)
	require.NoError(t, os.Mkdir(filepath.Join(root, "yes"), 0o755))
	path, err := lib.Resolve("yes")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "yes"), path)
}
