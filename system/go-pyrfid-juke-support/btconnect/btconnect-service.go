package btconnect

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const DEFAULT_ASLA_CONFIG = "/home/pi/.asoundrc"
const DEFAULT_PROJECT_ROOT = "/home/pi/workspace/pyrfid-jukebox"
const DEFAULT_RELATIVE_ASLA_CONFIG = "/system/home/.asoundrc"

type CommandExecutor interface {
	Command(name string, arg ...string) Cmd
}

type Cmd interface {
	Run() error
}

type OSCommandExecutor struct{}

func (e *OSCommandExecutor) Command(name string, arg ...string) Cmd {
	return exec.Command(name, arg...)
}

type BtConnectService struct {
	cmdExecutor CommandExecutor
}

func NewBtConnectService(cmdExecutor CommandExecutor) *BtConnectService {
	return &BtConnectService{
		cmdExecutor: cmdExecutor,
	}
}

func (bt *BtConnectService) Run() error {
	device := os.Getenv("PJ_BLUETOOTH_DEVICE")
	if device == "" {
		return fmt.Errorf("env var PJ_BLUETOOTH_DEVICE not set")
	}

	err := bt.updateALSAConfig()
	if err != nil {
		log.Printf("Error updating ALSA config: %v", err)
		return err
	}

	count, err := bt.getBluetoothConnectionCount(device)
	if err != nil {
		log.Printf("Error: %v", err)
		return err
	}
	log.Printf("Number of connections: %d\n", count)

	return nil
}

func (bt *BtConnectService) getBluetoothConnectionCount(device string) (int, error) {
	cmd := exec.Command("bluetoothctl", "info", device)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0, err
	}

	return bt.countConnectedLines(out.String()), nil
}

func (*BtConnectService) countConnectedLines(output string) int {
	scanner := bufio.NewScanner(strings.NewReader(output))
	count := 0
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "Connected: yes") {
			count++
		}
	}
	return count
}

func (bt *BtConnectService) updateALSAConfig() error {
	if hasChanged, err := bt.hasALSAConfigChanged(); err != nil {
		return err
	} else if hasChanged {
		log.Println("ALSA config has changed. Copying over system config...")

		aslaConfig := bt.getALSASystemConfig()
		projectAslaConfig := bt.getALSARepoConfig()

		if err := bt.copyFile(aslaConfig, projectAslaConfig); err != nil {
			log.Printf("Error copying file: %v", err)
			return err
		}

		if err := bt.cmdExecutor.Command("sudo", "alsactl", "restore").Run(); err != nil {
			log.Printf("Error executing alsactl restore: %v", err)
			return err
		}
	}
	return nil
}

func (bt *BtConnectService) hasALSAConfigChanged() (bool, error) {
	aslaConfig := bt.getALSASystemConfig()
	projectAslaConfig := bt.getALSARepoConfig()

	if diff, err := bt.filesAreDifferent(aslaConfig, projectAslaConfig); err != nil {
		return false, err
	} else {
		return diff, nil
	}
}

func (*BtConnectService) copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (*BtConnectService) getALSASystemConfig() string {
	aslaConfig := os.Getenv("PJ_ALSA_CONFIG")
	if aslaConfig == "" {
		aslaConfig = DEFAULT_ASLA_CONFIG
	}
	return aslaConfig
}

func (*BtConnectService) getALSARepoConfig() string {
	projectRoot := os.Getenv("PJ_PROJECT_ROOT")
	if projectRoot == "" {
		projectRoot = DEFAULT_PROJECT_ROOT
	}
	return filepath.Join(projectRoot, DEFAULT_RELATIVE_ASLA_CONFIG)
}

func (*BtConnectService) filesAreDifferent(file1, file2 string) (bool, error) {
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
