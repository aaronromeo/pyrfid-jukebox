package btconnect

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"aaronromeo.com/soundsprout/bluetooth"
)

const DefaultSleepTime = 15

// ErrMissingDevice reports that PJ_BLUETOOTH_DEVICE is not set. It is
// fatal: the environment is loaded once at process start, so retrying
// cannot help.
var ErrMissingDevice = errors.New("env var PJ_BLUETOOTH_DEVICE not set")

// SleepFunc pauses for d, aborting early when ctx is done.
type SleepFunc func(ctx context.Context, d time.Duration) error

type Service struct {
	runner bluetooth.Runner
	logger *slog.Logger
	sleep  SleepFunc
}

// Option configures a Service.
type Option func(*Service)

// WithSleep replaces the service's sleep implementation (test seam).
func WithSleep(fn SleepFunc) Option {
	return func(s *Service) { s.sleep = fn }
}

// NewBtConnectService returns a Bluetooth-connection maintenance service.
func NewBtConnectService(runner bluetooth.Runner, logger *slog.Logger, opts ...Option) *Service {
	s := &Service{runner: runner, logger: logger, sleep: sleepCtx}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Run maintains the connection until ctx is canceled. A missing
// PJ_BLUETOOTH_DEVICE is fatal; connect failures are logged and
// retried.
func (bt *Service) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := bt.CheckAndConnect(); err != nil {
			if errors.Is(err, ErrMissingDevice) {
				return err
			}
			bt.logger.Warn("bluetooth connect failed", "error", err)
		}
		if err := bt.sleep(ctx, DefaultSleepTime*time.Second); err != nil {
			return err
		}
	}
}

// CheckAndConnect performs one maintenance cycle for the configured device.
func (bt *Service) CheckAndConnect() error {
	device := os.Getenv("PJ_BLUETOOTH_DEVICE")
	if device == "" {
		return ErrMissingDevice
	}
	connected, err := bluetooth.IsConnected(bt.runner, device)
	if err != nil {
		return err
	}
	if !connected {
		return bluetooth.Connect(bt.runner, device)
	}
	return nil
}
