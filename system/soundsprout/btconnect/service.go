package btconnect

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

const DefaultASLAConfig = "/home/pi/.asoundrc"
const DefaultProjectRoot = "/home/pi/workspace/pyrfid-jukebox"
const DefaultRelativeASLAConfig = "/system/home/.asoundrc"
const DefaultSleepTime = 15

type CommandExecutor interface {
	Command(name string, arg ...string) Cmd
}

type Cmd interface {
	Run() error
}

type OSCommandExecutor struct {
}

func (e *OSCommandExecutor) Command(name string, arg ...string) Cmd {
	return exec.Command(name, arg...)
}

type Service struct {
	cmdExecutor CommandExecutor
	logger      *slog.Logger
}

func NewBtConnectService(cmdExecutor CommandExecutor, logger *slog.Logger) *Service {
	return &Service{
		cmdExecutor: cmdExecutor,
		logger:      logger,
	}
}

func (bt *Service) Run() error {
	for {
		device := os.Getenv("PJ_BLUETOOTH_DEVICE")
		if device == "" {
			return fmt.Errorf("env var PJ_BLUETOOTH_DEVICE not set")
		}

		count, err := bt.getBluetoothConnectionCount(device)
		if err != nil {
			bt.logger.Error("bt.getBluetoothConnectionCount falure", "error", err)
			return err
		}
		bt.logger.Info("found connections", "connections", count)
		if count == 0 {
			bt.logger.Info(fmt.Sprintf("attempting to connect to the device %s\n", device))
			if err = bt.connectBluetoothConnection(device); err != nil {
				bt.logger.Error("bt.connectBluetoothConnection falure", "error", err)
				return err
			}
			bt.logger.Info("no connection errors")
		}

		bt.logger.Info(fmt.Sprintf("sleeping for %d seconds\n", DefaultSleepTime))
		time.Sleep(DefaultSleepTime * time.Second)
	}
}

func (bt *Service) getBluetoothConnectionCount(device string) (int, error) {
	cmd := exec.Command("bluetoothctl", "info", device)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0, err
	}

	return bt.countConnectedLines(out.String()), nil
}

func (bt *Service) connectBluetoothConnection(device string) error {
	cmd := exec.Command("bluetoothctl", "connect", device)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (*Service) countConnectedLines(output string) int {
	scanner := bufio.NewScanner(strings.NewReader(output))
	count := 0
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "Connected: yes") {
			count++
		}
	}
	return count
}
