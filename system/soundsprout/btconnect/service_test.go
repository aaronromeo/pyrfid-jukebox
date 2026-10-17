package btconnect_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"aaronromeo.com/soundsprout/btconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	outputs map[string]string
	runErrs map[string]error
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
	return f.runErrs[key]
}

// recordingSleep records delays and cancels the loop's context once
// stop sleeps have been recorded.
type recordingSleep struct {
	delays []time.Duration
	stop   int
	cancel context.CancelFunc
}

func (r *recordingSleep) sleep(ctx context.Context, d time.Duration) error {
	r.delays = append(r.delays, d)
	if len(r.delays) >= r.stop {
		r.cancel()
	}
	return ctx.Err()
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCheckAndConnectMissingEnv(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "")
	svc := btconnect.NewBtConnectService(&fakeRunner{}, nil)
	err := svc.CheckAndConnect()
	require.Error(t, err)
	assert.ErrorIs(t, err, btconnect.ErrMissingDevice)
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

func TestRunMissingEnvFatal(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &recordingSleep{stop: 1, cancel: cancel}
	svc := btconnect.NewBtConnectService(
		&fakeRunner{}, newDiscardLogger(), btconnect.WithSleep(s.sleep),
	)
	err := svc.Run(ctx)
	require.ErrorIs(t, err, btconnect.ErrMissingDevice)
	assert.Empty(t, s.delays)
}

func TestRunContinuesOnConnectFailure(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "AA:BB:CC:DD:EE:FF")
	runner := &fakeRunner{runErrs: map[string]error{
		"bluetoothctl connect AA:BB:CC:DD:EE:FF": errors.New("connect failed"),
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &recordingSleep{stop: 2, cancel: cancel}
	svc := btconnect.NewBtConnectService(
		runner, newDiscardLogger(), btconnect.WithSleep(s.sleep),
	)
	err := svc.Run(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Len(t, s.delays, 2)
	assert.Len(t, runner.runs, 2)
}
