package bluetooth_test

import (
	"testing"
	"time"

	"aaronromeo.com/soundsprout/bluetooth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	outputs map[string]string
	runs    []string
}

func (f *fakeRunner) Output(name string, args ...string) (string, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	return f.outputs[key], nil
}

func (f *fakeRunner) Run(name string, args ...string) error {
	key := name
	for _, a := range args {
		key += " " + a
	}
	f.runs = append(f.runs, key)
	return nil
}

func TestParseInfo(t *testing.T) {
	out := "Device AA:BB:CC:DD:EE:FF\n\tName: Kitchen Speaker\n" +
		"\tPaired: yes\n\tTrusted: yes\n\tConnected: yes\n"
	info := bluetooth.ParseInfo(out)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", info.MAC)
	assert.Equal(t, "Kitchen Speaker", info.Name)
	assert.True(t, info.Paired)
	assert.True(t, info.Trusted)
	assert.True(t, info.Connected)
}

func TestParseDevices(t *testing.T) {
	out := "Device AA:BB:CC:DD:EE:FF Kitchen Speaker\n" +
		"Device 11:22:33:44:55:66\n"
	devices := bluetooth.ParseDevices(out)
	require.Len(t, devices, 2)
	assert.Equal(t, "11:22:33:44:55:66", devices[0].MAC)
	assert.Equal(t, "Kitchen Speaker", devices[1].Name)
}

func TestValidMAC(t *testing.T) {
	assert.True(t, bluetooth.ValidMAC("AA:BB:CC:DD:EE:FF"))
	assert.False(t, bluetooth.ValidMAC("not-a-mac"))
}

func TestReconnectSkipsWhenConnected(t *testing.T) {
	r := &fakeRunner{outputs: map[string]string{
		"bluetoothctl info AA:BB:CC:DD:EE:FF": "Connected: yes\n",
	}}
	require.NoError(t, bluetooth.Reconnect(r, "AA:BB:CC:DD:EE:FF"))
	assert.Empty(t, r.runs)
}

func TestReconnectConnectsWhenNotConnected(t *testing.T) {
	r := &fakeRunner{outputs: map[string]string{
		"bluetoothctl info AA:BB:CC:DD:EE:FF": "Connected: no\n",
	}}
	require.NoError(t, bluetooth.Reconnect(r, "AA:BB:CC:DD:EE:FF"))
	assert.Equal(t, []string{"bluetoothctl connect AA:BB:CC:DD:EE:FF"}, r.runs)
}

func TestPairPairsTrustsConnects(t *testing.T) {
	r := &fakeRunner{outputs: map[string]string{}}
	require.NoError(t, bluetooth.Pair(r, "AA:BB:CC:DD:EE:FF"))
	assert.Equal(t, []string{
		"bluetoothctl pair AA:BB:CC:DD:EE:FF",
		"bluetoothctl trust AA:BB:CC:DD:EE:FF",
		"bluetoothctl connect AA:BB:CC:DD:EE:FF",
	}, r.runs)
}

func TestScanReturnsKnownDevices(t *testing.T) {
	r := &fakeRunner{outputs: map[string]string{
		"bluetoothctl devices": "Device AA:BB:CC:DD:EE:FF Speaker\n",
	}}
	devices, err := bluetooth.Scan(r, 3*time.Second)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", devices[0].MAC)
}
