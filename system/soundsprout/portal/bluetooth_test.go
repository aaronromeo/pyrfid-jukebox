package portal_test

import (
	"os"
	"path/filepath"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBluetooth(t *testing.T) (*portal.Bluetooth, *fakeRunner, *portal.Settings, string, string) {
	t.Helper()
	dir := t.TempDir()
	r := newFakeRunner()
	asoundrc := filepath.Join(dir, ".asoundrc")
	cmusrc := filepath.Join(dir, "cmus-rc")
	settings := portal.NewSettings(
		filepath.Join(dir, "conf"),
		filepath.Join(dir, "settings.lock"),
	)
	audio := portal.NewAudioConfig(asoundrc, cmusrc, r)
	return portal.NewBluetooth(r, settings, audio), r, settings, asoundrc, cmusrc
}

func TestBluetoothActivatePersistsAndApplies(t *testing.T) {
	bt, r, settings, asoundrc, cmusrc := newTestBluetooth(t)
	require.NoError(t, bt.Activate("AA:BB:CC:DD:EE:FF"))

	mac, err := settings.Get("PJ_BLUETOOTH_DEVICE")
	require.NoError(t, err)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", mac)

	asound, err := os.ReadFile(asoundrc)
	require.NoError(t, err)
	assert.Contains(t, string(asound), `device "AA:BB:CC:DD:EE:FF"`)

	cmus, err := os.ReadFile(cmusrc)
	require.NoError(t, err)
	assert.Contains(t, string(cmus), "bluealsa:DEV=AA:BB:CC:DD:EE:FF")

	assert.Contains(t, r.runs, "bluetoothctl connect AA:BB:CC:DD:EE:FF")
	assert.Contains(t, r.runs, "sudo supervisorctl restart btconnect")
}

func TestBluetoothActivateRejectsBadMAC(t *testing.T) {
	bt, _, _, _, _ := newTestBluetooth(t)
	require.Error(t, bt.Activate("nope"))
}
