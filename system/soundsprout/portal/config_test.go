package portal_test

import (
	"os"
	"path/filepath"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAudioConfigApplyWritesAndRestarts(t *testing.T) {
	dir := t.TempDir()
	asoundrc := filepath.Join(dir, ".asoundrc")
	cmusrc := filepath.Join(dir, "cmus-rc")
	r := newFakeRunner()
	audio := portal.NewAudioConfig(asoundrc, cmusrc, r)

	require.NoError(t, audio.Apply("AA:BB:CC:DD:EE:FF"))

	asound, err := os.ReadFile(asoundrc)
	require.NoError(t, err)
	assert.Contains(t, string(asound), `device "AA:BB:CC:DD:EE:FF"`)

	cmus, err := os.ReadFile(cmusrc)
	require.NoError(t, err)
	assert.Contains(t, string(cmus), "bluealsa:DEV=AA:BB:CC:DD:EE:FF")

	assert.Equal(t, []string{
		"sudo supervisorctl restart btconnect",
		"sudo supervisorctl restart cmus_manager",
		"sudo alsactl restore",
	}, r.runs)
}
