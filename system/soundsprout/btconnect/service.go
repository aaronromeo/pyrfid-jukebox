package btconnect

import (
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

type Service struct {
	runner bluetooth.Runner
	logger *slog.Logger
}

// NewBtConnectService returns a Bluetooth-connection maintenance service.
func NewBtConnectService(runner bluetooth.Runner, logger *slog.Logger) *Service {
	return &Service{runner: runner, logger: logger}
}

func (bt *Service) Run() error {
	for {
		if err := bt.CheckAndConnect(); err != nil {
			return err
		}
		time.Sleep(DefaultSleepTime * time.Second)
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
