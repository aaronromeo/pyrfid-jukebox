package portal_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexRendersHello(t *testing.T) {
	srv, err := portal.NewServer(nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Hello, world")
}

func TestUnknownPathIsNotFound(t *testing.T) {
	srv, err := portal.NewServer(nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
