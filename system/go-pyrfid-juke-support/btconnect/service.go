package btconnect

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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

		err := bt.updateALSAConfig()
		if err != nil {
			bt.logger.Error("updating ALSA falure", "error", err)
			return err
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

func (bt *Service) updateALSAConfig() error {
	if hasChanged, errHC := bt.hasALSAConfigChanged(); errHC != nil {
		return errHC
	} else if hasChanged {
		bt.logger.Info("ALSA config has changed. Copying over system config...")

		aslaConfig := bt.getALSASystemConfig()
		projectAslaConfig := bt.getALSARepoConfig()

		if err := bt.copyFile(aslaConfig, projectAslaConfig); err != nil {
			bt.logger.Error(
				"copy file error",
				"error", err,
				"aslaConfig", aslaConfig,
				"projectAslaConfig", projectAslaConfig,
			)
			return err
		}

		if err := bt.cmdExecutor.Command("sudo", "alsactl", "restore").Run(); err != nil {
			bt.logger.Error(
				"alsactl restore error",
				"error", err,
			)
			return err
		}
	}
	return nil
}

func (bt *Service) hasALSAConfigChanged() (bool, error) {
	aslaConfig := bt.getALSASystemConfig()
	projectAslaConfig := bt.getALSARepoConfig()

	var diff bool
	var err error
	if diff, err = bt.filesAreDifferent(aslaConfig, projectAslaConfig); err != nil {
		return false, err
	}
	return diff, nil
}

func (*Service) copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (*Service) getALSASystemConfig() string {
	aslaConfig := os.Getenv("PJ_ALSA_CONFIG")
	if aslaConfig == "" {
		aslaConfig = DefaultASLAConfig
	}
	return aslaConfig
}

func (*Service) getALSARepoConfig() string {
	projectRoot := os.Getenv("PJ_PROJECT_ROOT")
	if projectRoot == "" {
		projectRoot = DefaultProjectRoot
	}
	return filepath.Join(projectRoot, DefaultRelativeASLAConfig)
}

func (*Service) filesAreDifferent(file1, file2 string) (bool, error) {
	bytes1, err := os.ReadFile(file1)
	if err != nil {
		return false, err
	}

	bytes2, err := os.ReadFile(file2)
	if err != nil {
		return false, err
	}

	return !bytes.Equal(bytes1, bytes2), nil
}
