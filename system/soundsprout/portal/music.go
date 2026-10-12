package portal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Folder struct {
	Name     string
	Path     string
	MP3Count int
}

type Library struct {
	root string
}

// NewLibrary returns a Library that lists folders under root.
func NewLibrary(root string) *Library { return &Library{root: root} }

// Folders lists immediate subdirectories of the music root with their .mp3 counts.
func (l *Library) Folders() ([]Folder, error) {
	entries, err := os.ReadDir(l.root)
	if os.IsNotExist(err) {
		return []Folder{}, nil
	}
	if err != nil {
		return nil, err
	}
	var folders []Folder
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(l.root, e.Name())
		count, err := countMP3(path)
		if err != nil {
			return nil, err
		}
		folders = append(folders, Folder{Name: e.Name(), Path: path, MP3Count: count})
	}
	sort.Slice(folders, func(i, j int) bool { return folders[i].Name < folders[j].Name })
	return folders, nil
}

// Resolve confines name to the music directory and requires an existing dir.
func (l *Library) Resolve(name string) (string, error) {
	if name == "" || name == "." || name == ".." ||
		strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("invalid folder name %q", name)
	}
	abs, err := filepath.Abs(filepath.Join(l.root, name))
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(l.root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("folder %q escapes music directory", name)
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("folder %q not found", name)
	}
	return abs, nil
}

func countMP3(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".mp3") {
			n++
		}
	}
	return n, nil
}
