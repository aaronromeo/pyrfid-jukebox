# Web Portal Part 2 — RFID Card Manager Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Let the web portal list, assign, repoint, and remove RFID-card → music-folder mappings in `server/rfid_map.json`, and harden the Python writer so the portal and `player.py` never clobber each other.

**Architecture:** Two deep modules (`MapStore` for the JSON map, `Library` for the `music/` folder tree) sit behind HTTP handlers in the existing `portal` package. `player.py` is switched onto a new `server/rfid_map_store.py` that uses the exact same lock file and `flock(LOCK_EX)` semantics as the Go `MapStore`.

**Tech Stack:** Go stdlib (`net/http`, `html/template`), existing `testify`; Python 3 + `pytest`; GitHub Actions.

**Depends on:** `docs/superpowers/plans/2026-10-12-web-portal-1-framework.md` (the `portal` package and `serve` command must exist).

## Global Constraints

- Go module path: `aaronromeo.com/soundsprout`. Go floor: 1.21 (CI).
- No shell anywhere: always `exec.Command`. Never pass user input to a shell.
- Go tests live in external `_test` packages (e.g. `package portal_test`) — the `testpackage` linter is enabled in `system/soundsprout/.golangci.yml`.
- Canonical paths (used as defaults; tests inject temp paths):
  - music dir `/home/pi/workspace/pyrfid-jukebox/music`
  - map file `/home/pi/workspace/pyrfid-jukebox/server/rfid_map.json`
  - map lock `/tmp/pyrfid_jukebox.map.lock`
- JSON map format is `{ "<card_id>": "<abs_path>" }`, pretty-printed with 4-space indent (compatible with Python `json.dump(indent=4)`).
- `rfid_map.json` writes are atomic (temp file in same dir + `os.Rename`) and guarded by `flock(LOCK_EX)` on `/tmp/pyrfid_jukebox.map.lock` — the **same lock file** Python uses.
- Only `.mp3` files are counted/played (unchanged, issue #16).
- No authentication (LAN-only) — POSTs require same-host `Origin`/`Referer`.
- Run Go tests with `ASDF_GOLANG_VERSION=1.25.6` locally.
- Python: black line-length 79, flake8 max-line-length 80.

---

### Task 1: `portal.MapStore`

**Files:**
- Create: `system/soundsprout/portal/rfidmap.go`
- Create: `system/soundsprout/portal/fileutil.go`
- Test: `system/soundsprout/portal/rfidmap_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `func NewMapStore(path, lockPath string) *MapStore`
  - `func (m *MapStore) Load() (map[string]string, error)`
  - `func (m *MapStore) Update(mutate func(map[string]string) error) error`
  - `func (m *MapStore) Save(data map[string]string) error`
  - `func (m *MapStore) Assign(cardID, folder string) error`
  - `func (m *MapStore) Remove(cardID string) error`

- [x] **Step 1: Write the failing tests**

Create `system/soundsprout/portal/rfidmap_test.go`:

```go
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
```

- [x] **Step 2: Run tests to verify they fail**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/...`
Expected: FAIL — `undefined: portal.NewMapStore`.

- [x] **Step 3: Write `rfidmap.go`**

Create `system/soundsprout/portal/rfidmap.go`:

```go
package portal

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"
)

type MapStore struct {
	path     string
	lockPath string
}

// NewMapStore returns a store for the RFID-to-music map at path. All writes take
// an exclusive flock on lockPath, which Python's rfid_map_store uses too.
func NewMapStore(path, lockPath string) *MapStore {
	return &MapStore{path: path, lockPath: lockPath}
}

// Load reads the map. A missing or empty file is an empty map.
func (m *MapStore) Load() (map[string]string, error) {
	raw, err := os.ReadFile(m.path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read map: %w", err)
	}
	out := map[string]string{}
	if len(raw) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse map: %w", err)
	}
	return out, nil
}

// Update runs mutate inside an exclusive lock and persists the result atomically.
func (m *MapStore) Update(mutate func(map[string]string) error) error {
	lock, err := os.OpenFile(m.lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open lock: %w", err)
	}
	defer func() { _ = lock.Close() }()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock: %w", err)
	}
	defer func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) }()

	data, err := m.Load()
	if err != nil {
		return err
	}
	if err := mutate(data); err != nil {
		return err
	}
	return m.writeAtomic(data)
}

// Save replaces the whole map with data.
func (m *MapStore) Save(data map[string]string) error {
	return m.Update(func(current map[string]string) error {
		for k := range current {
			delete(current, k)
		}
		for k, v := range data {
			current[k] = v
		}
		return nil
	})
}

// Assign points cardID at folder.
func (m *MapStore) Assign(cardID, folder string) error {
	return m.Update(func(data map[string]string) error {
		data[cardID] = folder
		return nil
	})
}

// Remove deletes cardID from the map.
func (m *MapStore) Remove(cardID string) error {
	return m.Update(func(data map[string]string) error {
		delete(data, cardID)
		return nil
	})
}

func (m *MapStore) writeAtomic(data map[string]string) error {
	buf, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	return writeFileAtomic(m.path, string(buf))
}
```

- [x] **Step 4: Write the shared atomic-write helper**

Create `system/soundsprout/portal/fileutil.go`:

```go
package portal

import (
	"os"
	"path/filepath"
)

// writeFileAtomic writes content to a temp file in path's directory, fsyncs it,
// and renames it over path.
func writeFileAtomic(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
```

- [x] **Step 5: Run tests to verify they pass**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/...`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add system/soundsprout/portal/rfidmap.go system/soundsprout/portal/fileutil.go \
  system/soundsprout/portal/rfidmap_test.go
git commit -m "feat(portal): add atomic, locked map store"
```

---

### Task 2: `portal.Library`

**Files:**
- Create: `system/soundsprout/portal/music.go`
- Test: `system/soundsprout/portal/music_test.go`

**Interfaces:**
- Produces:
  - `type Folder struct { Name, Path string; MP3Count int }`
  - `func NewLibrary(root string) *Library`
  - `func (l *Library) Folders() ([]Folder, error)`
  - `func (l *Library) Resolve(name string) (string, error)`

- [x] **Step 1: Write the failing tests**

Create `system/soundsprout/portal/music_test.go`:

```go
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
```

- [x] **Step 2: Run tests to verify they fail**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/... -run TestLibrary`
Expected: FAIL — `undefined: portal.NewLibrary`.

- [x] **Step 3: Write `music.go`**

Create `system/soundsprout/portal/music.go`:

```go
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
```

- [x] **Step 4: Run tests to verify they pass**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/...`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add system/soundsprout/portal/music.go system/soundsprout/portal/music_test.go
git commit -m "feat(portal): add music library folder listing"
```

---

### Task 3: Cards HTTP handlers, templates, and service wiring

**Files:**
- Modify: `system/soundsprout/portal/server.go` (replace entire file)
- Modify: `system/soundsprout/portal/service.go` (replace entire file)
- Modify: `system/soundsprout/portal/templates/index.html` (add nav link)
- Create: `system/soundsprout/portal/templates/cards.html`
- Modify: `system/soundsprout/portal/server_test.go` (replace entire file)
- Modify: `README.md`, `AGENTS.md`

**Interfaces:**
- Consumes: `MapStore`, `Library`.
- Produces:
  - `func NewServer(mapStore *MapStore, library *Library, logger *slog.Logger) (*Server, error)`
  - `func NewService(logger *slog.Logger) (*Server, error)` (signature unchanged)
  - `type Config struct{...}`; `func DefaultConfig() Config`
  - Routes: `GET /`, `GET /cards`, `POST /cards/assign`, `POST /cards/remove`.

- [x] **Step 1: Write the failing tests**

Replace the entire contents of `system/soundsprout/portal/server_test.go` with:

```go
package portal_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) (*portal.Server, *portal.MapStore, string) {
	t.Helper()
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	require.NoError(t, os.MkdirAll(filepath.Join(music, "card-1"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(music, "card-1", "a.mp3"), []byte("x"), 0o644))

	mapStore := portal.NewMapStore(
		filepath.Join(dir, "rfid_map.json"),
		filepath.Join(dir, "map.lock"),
	)
	library := portal.NewLibrary(music)
	srv, err := portal.NewServer(mapStore, library, nil)
	require.NoError(t, err)
	return srv, mapStore, music
}

func postForm(t *testing.T, h http.Handler, path string, form url.Values, host string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://"+host)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestIndexRendersHello(t *testing.T) {
	srv, _, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Hello, world")
}

func TestUnknownPathIsNotFound(t *testing.T) {
	srv, _, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAssignAssignsKnownCard(t *testing.T) {
	srv, mapStore, music := newTestServer(t)
	require.NoError(t, mapStore.Assign("999", ""))

	rec := postForm(t, srv.Routes(), "/cards/assign",
		url.Values{"card_id": {"999"}, "folder": {"card-1"}}, "example.com")

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	data, err := mapStore.Load()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(music, "card-1"), data["999"])
}

func TestAssignRejectsUnknownCard(t *testing.T) {
	srv, _, _ := newTestServer(t)

	rec := postForm(t, srv.Routes(), "/cards/assign",
		url.Values{"card_id": {"missing"}, "folder": {"card-1"}}, "example.com")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRemoveDeletesCard(t *testing.T) {
	srv, mapStore, _ := newTestServer(t)
	require.NoError(t, mapStore.Assign("999", "/x"))

	rec := postForm(t, srv.Routes(), "/cards/remove",
		url.Values{"card_id": {"999"}}, "example.com")

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	data, err := mapStore.Load()
	require.NoError(t, err)
	_, ok := data["999"]
	assert.False(t, ok)
}

func TestCSRFGuardRejectsCrossOrigin(t *testing.T) {
	srv, mapStore, _ := newTestServer(t)
	require.NoError(t, mapStore.Assign("999", ""))

	rec := postForm(t, srv.Routes(), "/cards/assign",
		url.Values{"card_id": {"999"}, "folder": {"card-1"}}, "evil.com")

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/...`
Expected: FAIL — `too many arguments in call to portal.NewServer`.

- [x] **Step 3: Replace `server.go`**

Replace the entire contents of `system/soundsprout/portal/server.go` with:

```go
package portal

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"os"
)

//go:embed templates/*.html
var templatesFS embed.FS

type cardRow struct {
	ID       string
	Path     string
	Exists   bool
	MP3Count int
}

type Server struct {
	mapStore *MapStore
	library  *Library
	logger   *slog.Logger
	tmpl     *template.Template
}

// NewServer parses the embedded templates and returns a Server.
func NewServer(mapStore *MapStore, library *Library, logger *slog.Logger) (*Server, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{mapStore: mapStore, library: library, logger: logger, tmpl: tmpl}, nil
}

// Routes returns the portal's HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/cards", s.handleCards)
	mux.HandleFunc("/cards/assign", s.handleAssign)
	mux.HandleFunc("/cards/remove", s.handleRemove)
	return s.csrfGuard(mux)
}

// csrfGuard rejects POSTs whose Origin/Referer host differs from the request host.
func (s *Server) csrfGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = r.Header.Get("Referer")
			}
			if origin != "" {
				if u, err := url.Parse(origin); err != nil || u.Host != r.Host {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil && s.logger != nil {
		s.logger.Error("render", "template", name, "error", err)
	}
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	target := r.Referer()
	if target == "" {
		target = "/cards"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.render(w, "index.html", map[string]any{"Title": "SoundSprout"})
}

func (s *Server) handleCards(w http.ResponseWriter, r *http.Request) {
	data, err := s.mapStore.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	folders, err := s.library.Folders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows := make([]cardRow, 0, len(data))
	unassigned := []string{}
	for id, path := range data {
		if path == "" {
			unassigned = append(unassigned, id)
			continue
		}
		info, statErr := os.Stat(path)
		exists := statErr == nil && info.IsDir()
		count := 0
		if exists {
			count, _ = countMP3(path)
		}
		rows = append(rows, cardRow{ID: id, Path: path, Exists: exists, MP3Count: count})
	}
	s.render(w, "cards.html", map[string]any{
		"Rows":       rows,
		"Unassigned": unassigned,
		"Folders":    folders,
	})
}

func (s *Server) handleAssign(w http.ResponseWriter, r *http.Request) {
	cardID := r.FormValue("card_id")
	folder := r.FormValue("folder")

	data, err := s.mapStore.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, ok := data[cardID]; !ok {
		http.Error(w, "unknown card id", http.StatusBadRequest)
		return
	}
	folderPath, err := s.library.Resolve(folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.mapStore.Assign(cardID, folderPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.redirect(w, r)
}

func (s *Server) handleRemove(w http.ResponseWriter, r *http.Request) {
	if err := s.mapStore.Remove(r.FormValue("card_id")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.redirect(w, r)
}
```

- [x] **Step 4: Replace `service.go`**

Replace the entire contents of `system/soundsprout/portal/service.go` with:

```go
package portal

import "log/slog"

const (
	DefaultMusicDir     = "/home/pi/workspace/pyrfid-jukebox/music"
	DefaultMapFile      = "/home/pi/workspace/pyrfid-jukebox/server/rfid_map.json"
	DefaultMapLock      = "/tmp/pyrfid_jukebox.map.lock"
	DefaultSettingsFile = "/home/pi/.soundsprout/conf"
	DefaultSettingsLock = "/tmp/pyrfid_jukebox.settings.lock"
	DefaultAsoundrc     = "/home/pi/.asoundrc"
	DefaultCmusRc       = "/home/pi/.config/cmus/rc"
)

// Config holds every injectable path the portal uses.
type Config struct {
	MusicDir     string
	MapFile      string
	MapLock      string
	SettingsFile string
	SettingsLock string
	AsoundrcPath string
	CmusRcPath   string
}

// DefaultConfig returns the canonical on-device paths.
func DefaultConfig() Config {
	return Config{
		MusicDir:     DefaultMusicDir,
		MapFile:      DefaultMapFile,
		MapLock:      DefaultMapLock,
		SettingsFile: DefaultSettingsFile,
		SettingsLock: DefaultSettingsLock,
		AsoundrcPath: DefaultAsoundrc,
		CmusRcPath:   DefaultCmusRc,
	}
}

// NewService wires the portal's dependencies and returns a ready Server.
func NewService(logger *slog.Logger) (*Server, error) {
	cfg := DefaultConfig()
	mapStore := NewMapStore(cfg.MapFile, cfg.MapLock)
	library := NewLibrary(cfg.MusicDir)
	return NewServer(mapStore, library, logger)
}
```

- [x] **Step 5: Update `index.html`**

Replace the entire contents of `system/soundsprout/portal/templates/index.html` with:

```html
<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>{{.Title}}</title></head>
<body>
<h1>{{.Title}}</h1>
<p>Hello, world.</p>
<p><a href="/cards">Cards</a></p>
</body>
</html>
```

- [x] **Step 6: Create `cards.html`**

Create `system/soundsprout/portal/templates/cards.html`:

```html
<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Cards</title></head>
<body>
<h1>Cards</h1>
<p><a href="/">Home</a></p>

<h2>Unassigned cards</h2>
{{if .Unassigned}}
<table>
  <tr><th>Card</th><th>Assign folder</th></tr>
  {{range .Unassigned}}
  <tr>
    <td>{{.}}</td>
    <td>
      <form method="post" action="/cards/assign">
        <input type="hidden" name="card_id" value="{{.}}">
        <select name="folder">
          {{range $.Folders}}<option value="{{.Name}}">{{.Name}} ({{.MP3Count}} mp3)</option>{{end}}
        </select>
        <button type="submit">Assign</button>
      </form>
    </td>
  </tr>
  {{end}}
</table>
{{else}}<p>None.</p>{{end}}

<h2>Mappings</h2>
{{if .Rows}}
<table>
  <tr><th>Card</th><th>Folder</th><th>Status</th><th></th></tr>
  {{range .Rows}}
  <tr>
    <td>{{.ID}}</td>
    <td>{{.Path}}</td>
    <td>{{if .Exists}}{{.MP3Count}} mp3{{else}}missing{{end}}</td>
    <td>
      <form method="post" action="/cards/remove">
        <input type="hidden" name="card_id" value="{{.ID}}">
        <button type="submit">Remove</button>
      </form>
    </td>
  </tr>
  {{end}}
</table>
{{else}}<p>No mappings yet.</p>{{end}}
</body>
</html>
```

- [x] **Step 7: Run tests to verify they pass**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/...`
Expected: PASS (all card + map + library tests).

- [x] **Step 8: Document the Cards tab**

In `README.md`, append this subsection under the `## Web Portal` section:

```markdown
### Cards

The Cards tab (`/cards`) lists every known RFID card and its mapped folder, and offers a
folder picker over `music/` to assign cards that were seen but never mapped. Mappings are
stored in `server/rfid_map.json` (writes are atomic and take the same lock as `player.py`).
```

- [x] **Step 9: Commit**

```bash
git add system/soundsprout/portal/server.go system/soundsprout/portal/service.go \
  system/soundsprout/portal/server_test.go system/soundsprout/portal/templates README.md
git commit -m "feat(portal): add RFID card manager UI"
```

---

### Task 4: Python map store, pytest, and CI

**Files:**
- Create: `server/rfid_map_store.py`
- Create: `server/tests/__init__.py` (empty)
- Create: `server/tests/conftest.py`
- Create: `server/tests/test_rfid_map_store.py`
- Modify: `pyproject.toml`
- Modify: `requirements.txt`
- Modify: `.github/workflows/testing.yml`
- Modify: `AGENTS.md`

**Interfaces:**
- Produces: `load_map(path) -> dict`; `save_map_atomic(path, data) -> None`; `update_map(path, mutate, lock_path=MAP_LOCK) -> Any`.

- [x] **Step 1: Write the failing tests**

Create an empty `server/tests/__init__.py`.

Create `server/tests/conftest.py`:

```python
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
```

Create `server/tests/test_rfid_map_store.py`:

```python
import multiprocessing

from rfid_map_store import load_map, save_map_atomic, update_map


def test_load_missing_returns_empty(tmp_path):
    assert load_map(str(tmp_path / "nope.json")) == {}


def test_save_then_load_roundtrip(tmp_path):
    path = str(tmp_path / "map.json")
    save_map_atomic(path, {"1": "/a"})
    assert load_map(path) == {"1": "/a"}


def test_save_writes_indented_json(tmp_path):
    path = tmp_path / "map.json"
    save_map_atomic(str(path), {"1": "/a"})
    assert '    "1": "/a"' in path.read_text()


def test_update_mutates_and_persists(tmp_path):
    path = str(tmp_path / "map.json")
    save_map_atomic(path, {"1": "/a"})

    def mutate(data):
        data["2"] = "/b"
        return "ok"

    assert update_map(path, mutate, lock_path=str(tmp_path / "lock")) == "ok"
    assert load_map(path) == {"1": "/a", "2": "/b"}


def _add_key(args):
    path, lock_path, key = args
    update_map(path, lambda d: d.__setitem__(key, "/x"), lock_path=lock_path)


def test_update_is_safe_across_processes(tmp_path):
    path = str(tmp_path / "map.json")
    lock = str(tmp_path / "lock")
    save_map_atomic(path, {})
    procs = [
        multiprocessing.Process(target=_add_key, args=((path, lock, str(i)),))
        for i in range(5)
    ]
    for p in procs:
        p.start()
    for p in procs:
        p.join()
    assert len(load_map(path)) == 5
```

- [x] **Step 2: Run tests to verify they fail**

Run: `pip install pytest==8.3.4 && python3 -m pytest server/tests -q`
Expected: FAIL — `ModuleNotFoundError: No module named 'rfid_map_store'`.

- [x] **Step 3: Write `server/rfid_map_store.py`**

Create `server/rfid_map_store.py`:

```python
"""Atomic, lock-protected I/O for the RFID-to-music map.

The lock file and flock(LOCK_EX) semantics are shared with the Go portal
(see system/soundsprout/portal/rfidmap.go). Do not change the lock path
without changing the Go side too.
"""

import fcntl
import json
import os
import tempfile

MAP_LOCK = "/tmp/pyrfid_jukebox.map.lock"


def load_map(path):
    if not os.path.exists(path):
        return {}
    with open(path) as handle:
        contents = handle.read()
    if not contents:
        return {}
    return json.loads(contents)


def save_map_atomic(path, data):
    directory = os.path.dirname(os.path.abspath(path))
    os.makedirs(directory, exist_ok=True)
    fd, tmp = tempfile.mkstemp(dir=directory, prefix=".rfid_map-", suffix=".tmp")
    try:
        with os.fdopen(fd, "w") as handle:
            json.dump(data, handle, indent=4)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(tmp, path)
    except BaseException:
        try:
            os.unlink(tmp)
        except OSError:
            pass
        raise


def update_map(path, mutate, lock_path=MAP_LOCK):
    with open(lock_path, "a+") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        try:
            data = load_map(path)
            result = mutate(data)
            save_map_atomic(path, data)
        finally:
            fcntl.flock(lock, fcntl.LOCK_UN)
    return result
```

- [x] **Step 4: Add pytest config + deps**

Append to `pyproject.toml`:

```toml
[tool.pytest.ini_options]
testpaths = ["server/tests"]
```

Add to `requirements.txt` (keep alphabetical placement):

```text
iniconfig==2.0.0
pluggy==1.5.0
pytest==8.3.4
```

- [x] **Step 5: Run tests to verify they pass**

Run: `pip install pytest==8.3.4 && python3 -m pytest -q`
Expected: all tests PASS (config in `pyproject.toml` points at `server/tests`).

- [x] **Step 6: Add the CI job**

In `.github/workflows/testing.yml`, add this job after the existing ones:

```yaml
  python:
    name: Run Python tests
    runs-on: ubuntu-latest
    steps:
      - name: Check out code
        uses: actions/checkout@v2
      - name: Set up Python
        uses: actions/setup-python@v2
        with:
          python-version: "3.x"
      - name: Install pytest
        run: pip install pytest==8.3.4
      - name: Run tests
        run: python -m pytest -q
```

- [x] **Step 7: Document the Python tests in `AGENTS.md`**

In the `**Python:**` bullet list under `## Build, test, lint, format`, add:

```markdown
- Tests: `python3 -m pytest` (config in `pyproject.toml`, tests under `server/tests/`)
```

- [x] **Step 8: Commit**

```bash
git add server/rfid_map_store.py server/tests pyproject.toml requirements.txt \
  .github/workflows/testing.yml AGENTS.md
git commit -m "feat(server): atomic locked rfid map store with pytest coverage"
```

---

### Task 5: Wire `player.py` onto the map store

**Files:**
- Modify: `server/player.py`

**Interfaces:**
- Consumes: `rfid_map_store.load_map`, `rfid_map_store.save_map_atomic`, `rfid_map_store.update_map`.

- [x] **Step 1: Add the import**

In `server/player.py`, after `from logger import Logger` (line ~13) add:

```python
import rfid_map_store
```

- [x] **Step 2: Remove the unused `json` import**

Delete the `import json` line (line ~11). It is no longer referenced after the changes below.

- [x] **Step 3: Delete the non-atomic writer**

Delete the `data_to_map` function:

```python
def data_to_map(data):
    with open(RFID_TO_MUSIC_MAP, "w") as file:
        Logger.info(f"Writing to map file {RFID_TO_MUSIC_MAP}")
        json.dump(data, file, indent=4)
```

- [x] **Step 4: Replace the create-if-missing call**

Replace:

```python
            if not os.path.exists(RFID_TO_MUSIC_MAP):
                data_to_map(data)
```

with:

```python
            if not os.path.exists(RFID_TO_MUSIC_MAP):
                rfid_map_store.save_map_atomic(RFID_TO_MUSIC_MAP, {})
```

- [x] **Step 5: Replace the map read**

Replace:

```python
            with open(RFID_TO_MUSIC_MAP, "r") as file:
                data = json.load(file)
```

with:

```python
            data = rfid_map_store.load_map(RFID_TO_MUSIC_MAP)
```

- [x] **Step 6: Replace the unknown-card write**

Delete the `update_map = False` line (near line ~132) and the trailing block:

```python
            if update_map:
                data_to_map(data)
```

Then, in the unknown-card `else` branch, replace:

```python
                update_map = True
                data[rfid_id] = ""
```

with:

```python
                rfid_map_store.update_map(
                    RFID_TO_MUSIC_MAP,
                    lambda mapping: mapping.__setitem__(rfid_id, ""),
                )
```

- [x] **Step 7: Byte-compile to verify syntax**

Run: `python3 -m py_compile server/player.py`
Expected: exit 0.

- [x] **Step 8: Run flake8 + black checks**

Run: `black --check server/player.py server/rfid_map_store.py server/tests && flake8 server/player.py server/rfid_map_store.py server/tests`
Expected: no changes needed, no lint errors.

- [x] **Step 9: Commit**

```bash
git add server/player.py
git commit -m "fix(server): use atomic locked map writes in player"
```

---

### Task 6: Final validation for Part 2

- [x] **Step 1: Run the full Go suite**

Run: `ASDF_GOLANG_VERSION=1.25.6 make test`
Expected: all packages PASS, including `portal`.

- [x] **Step 2: Run golangci-lint (if available)**

Run: `cd system/soundsprout && golangci-lint run --config .golangci.yml ./...`
Expected: no issues. (If not installed, skip; fix any findings the CI job reports.)

- [x] **Step 3: Run Python tests + lint**

Run: `python3 -m pytest -q && pre-commit run --all-files`
Expected: tests PASS; hooks clean.

- [x] **Step 4: Smoke-test the cards page**

Run:

```bash
cd system/soundsprout
ASDF_GOLANG_VERSION=1.25.6 go build -o /tmp/soundsprout-server ./cmd
/tmp/soundsprout-server serve --addr 127.0.0.1:8080 &
sleep 1
curl -sS http://127.0.0.1:8080/cards | grep -q "Cards" && echo "OK"
kill %1
```

Expected: prints `OK`.

- [x] **Step 5: Commit any final fixups**

```bash
git add -A
git commit -m "chore: part 2 card manager validation fixes"
```
