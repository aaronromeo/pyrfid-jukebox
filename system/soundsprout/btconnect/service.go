package btconnect

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"aaronromeo.com/soundsprout/bluetooth"
)

const DefaultSleepTime = 15

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
		return fmt.Errorf("env var PJ_BLUETOOTH_DEVICE not set")
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
