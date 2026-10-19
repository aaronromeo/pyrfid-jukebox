package portal_test

import (
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
)

// Both jukebox hosts serve music from ~/mp3s; the old music/ dir is abandoned.
func TestDefaultConfigMusicDirIsMP3Library(t *testing.T) {
	assert.Equal(t, "/home/pi/mp3s", portal.DefaultConfig().MusicDir)
}
