package btconnect_test

import (
	"bytes"
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
// stop sleeps have been recorded. onSleep (optional) runs after each
// record, receiving the number of sleeps so far.
type recordingSleep struct {
	delays  []time.Duration
	stop    int
	cancel  context.CancelFunc
	onSleep func(n int)
}

func (r *recordingSleep) sleep(ctx context.Context, d time.Duration) error {
	r.delays = append(r.delays, d)
	if r.onSleep != nil {
		r.onSleep(len(r.delays))
	}
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

func TestRunBackoffSequence(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "AA:BB:CC:DD:EE:FF")
	runner := &fakeRunner{runErrs: map[string]error{
		"bluetoothctl connect AA:BB:CC:DD:EE:FF": errors.New("connect failed"),
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	s := &recordingSleep{stop: 6, cancel: cancel}
	svc := btconnect.NewBtConnectService(
		runner, logger, btconnect.WithSleep(s.sleep),
	)
	err := svc.Run(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []time.Duration{
		15 * time.Second,
		30 * time.Second,
		60 * time.Second,
		120 * time.Second,
		240 * time.Second,
		300 * time.Second,
	}, s.delays)
	assert.Len(t, runner.runs, 6)
	assert.Contains(t, logs.String(), "retry_in")
}

func TestRunResetsBackoffOnSuccess(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "AA:BB:CC:DD:EE:FF")
	const connectKey = "bluetoothctl connect AA:BB:CC:DD:EE:FF"
	runner := &fakeRunner{runErrs: map[string]error{
		connectKey: errors.New("connect failed"),
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	s := &recordingSleep{stop: 4, cancel: cancel, onSleep: func(n int) {
		switch n {
		case 2:
			delete(runner.runErrs, connectKey) // cycle 3 succeeds
		case 3:
			runner.runErrs[connectKey] = errors.New("connect failed") // cycle 4 fails
		}
	}}
	svc := btconnect.NewBtConnectService(runner, logger, btconnect.WithSleep(s.sleep))
	err := svc.Run(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []time.Duration{
		15 * time.Second,
		30 * time.Second,
		15 * time.Second,
		15 * time.Second,
	}, s.delays)
	assert.Contains(t, logs.String(), "bluetooth reconnected")
	assert.Contains(t, logs.String(), "failures=2")
}
