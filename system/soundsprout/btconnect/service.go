package btconnect

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"aaronromeo.com/soundsprout/bluetooth"
)

const (
	DefaultSleepTime = 15
	maxSleepTime     = 300
)

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
// retried with exponential backoff.
func (bt *Service) Run(ctx context.Context) error {
	failures := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := bt.CheckAndConnect()
		if err != nil {
			if errors.Is(err, ErrMissingDevice) {
				return err
			}
			failures++
		} else if failures > 0 {
			bt.logger.Info("bluetooth reconnected", "failures", failures)
			failures = 0
		}
		delay := nextDelay(failures)
		if err != nil {
			bt.logger.Warn(
				"bluetooth connect failed",
				"error", err,
				"failures", failures,
				"retry_in", delay,
			)
		}
		if sleepErr := bt.sleep(ctx, delay); sleepErr != nil {
			return sleepErr
		}
	}
}

// nextDelay returns the backoff after n consecutive failures:
// 15s, 30s, 60s, 120s, 240s, then capped at 300s. n <= 0 returns the
// base interval.
func nextDelay(failures int) time.Duration {
	d := DefaultSleepTime * time.Second
	for i := 1; i < failures && d < maxSleepTime*time.Second; i++ {
		d *= 2
	}
	if d > maxSleepTime*time.Second {
		return maxSleepTime * time.Second
	}
	return d
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
