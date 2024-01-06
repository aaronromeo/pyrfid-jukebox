package btconnect

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	helper "aaronromeo.com/rfid-jukebox/system/bin/helper"
)

type Service struct {
	cmdExecutor       helper.CommandExecutor
	alsaConfigUpdater helper.ALSAConfigUpdater
}

func NewBtConnectService(cmdExecutor helper.CommandExecutor, alsaConfigUpdater helper.ALSAConfigUpdater) *Service {
	return &Service{
		cmdExecutor:       cmdExecutor,
		alsaConfigUpdater: alsaConfigUpdater,
	}
}

func (bt *Service) Run() error {
	device := os.Getenv("PJ_BLUETOOTH_DEVICE")
	if device == "" {
		return fmt.Errorf("env var PJ_BLUETOOTH_DEVICE not set")
	}

	err := bt.alsaConfigUpdater.UpdateALSAConfig(bt.cmdExecutor)
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

func (bt *Service) getBluetoothConnectionCount(device string) (int, error) {
	cmd := bt.cmdExecutor.Command("bluetoothctl", "info", device)
	err := cmd.Run()
	if err != nil {
		return 0, err
	}

	return bt.countConnectedLines(bt.cmdExecutor.GetOutput()), nil
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
