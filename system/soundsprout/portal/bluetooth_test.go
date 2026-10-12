package portal_test

import (
	"path/filepath"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBluetooth(t *testing.T) (*portal.Bluetooth, *fakeRunner, *portal.Settings) {
	t.Helper()
	dir := t.TempDir()
	r := newFakeRunner()
	settings := portal.NewSettings(
		filepath.Join(dir, "conf"),
		filepath.Join(dir, "settings.lock"),
	)
	audio := portal.NewAudioConfig(
		filepath.Join(dir, ".asoundrc"),
		filepath.Join(dir, "cmus-rc"),
		r,
	)
	return portal.NewBluetooth(r, settings, audio), r, settings
}

func TestBluetoothActivatePersistsAndApplies(t *testing.T) {
	bt, r, settings := newTestBluetooth(t)
	require.NoError(t, bt.Activate("AA:BB:CC:DD:EE:FF"))

	mac, err := settings.Get("PJ_BLUETOOTH_DEVICE")
	require.NoError(t, err)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", mac)

	assert.Contains(t, r.runs, "bluetoothctl connect AA:BB:CC:DD:EE:FF")
	assert.Contains(t, r.runs, "sudo supervisorctl restart btconnect")
}

func TestBluetoothActivateRejectsBadMAC(t *testing.T) {
	bt, _, _ := newTestBluetooth(t)
	require.Error(t, bt.Activate("nope"))
}
