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
