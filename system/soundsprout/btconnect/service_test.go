package btconnect_test

import (
	"testing"

	"aaronromeo.com/soundsprout/btconnect"
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

func TestCheckAndConnectMissingEnv(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "")
	svc := btconnect.NewBtConnectService(&fakeRunner{}, nil)
	require.Error(t, svc.CheckAndConnect())
}

func TestCheckAndConnectConnects(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "AA:BB:CC:DD:EE:FF")
	r := &fakeRunner{outputs: map[string]string{
		"bluetoothctl info AA:BB:CC:DD:EE:FF": "Connected: no\n",
	}}
	svc := btconnect.NewBtConnectService(r, nil)
	require.NoError(t, svc.CheckAndConnect())
	assert.Equal(t, []string{"bluetoothctl connect AA:BB:CC:DD:EE:FF"}, r.runs)
}
