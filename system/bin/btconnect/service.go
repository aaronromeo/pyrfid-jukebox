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
	CmdExecutor       helper.CommandExecutor
	AlsaConfigUpdater helper.ALSAConfigUpdater
}

func NewBtConnectService(cmdExecutor helper.CommandExecutor, alsaConfigUpdater helper.ALSAConfigUpdater) *Service {
	return &Service{
		CmdExecutor:       cmdExecutor,
		AlsaConfigUpdater: alsaConfigUpdater,
	}
}

func (bt *Service) Run() error {
	device := os.Getenv("PJ_BLUETOOTH_DEVICE")
	if device == "" {
		return fmt.Errorf("env var PJ_BLUETOOTH_DEVICE not set")
	}

	err := bt.AlsaConfigUpdater.UpdateALSAConfig(bt.CmdExecutor)
	if err != nil {
		log.Printf("Error updating ALSA config: %v", err)
		return err
	}

	count, err := bt.getBluetoothConnectionCount(device)
	if err != nil {
		log.Printf("Error: %v", err)
		return err
	}
	// TODO: The count should be greater than 0. This reconnect if the count is 0.
	log.Printf("Number of connections: %d\n", count)

	/*
		$ bluealsa-aplay -l
		**** List of PLAYBACK Bluetooth Devices ****
		hci0: 88:C6:26:23:95:3F [UE MINI BOOM], trusted audio-card
		A2DP (SBC): S16_LE 2 channels 48000 Hz
		**** List of CAPTURE Bluetooth Devices ****
	*/

	/*
		$ sudo service bluealsa status
		● bluealsa.service - BlueALSA service
			Loaded: loaded (/lib/systemd/system/bluealsa.service; enabled; vendor preset: enabled)
			Active: active (running) since Thu 2024-01-11 20:17:33 EST; 1 day 23h ago
			Docs: man:bluealsa(8)
		Main PID: 425 (bluealsa)
			Tasks: 6 (limit: 414)
				CPU: 11min 36.250s
			CGroup: /system.slice/bluealsa.service
					└─425 /usr/bin/bluealsa -S -p a2dp-source -p a2dp-sink

		Jan 11 20:17:30 normajean systemd[1]: Starting BlueALSA service...
		Jan 11 20:17:33 normajean systemd[1]: Started BlueALSA service.
	*/

	/*
		$ bluetoothctl show
		Controller B8:27:EB:B0:1C:17 (public)
			Name: normajean
			Alias: normajean
			Class: 0x000c0000
			Powered: yes
			Discoverable: no
			DiscoverableTimeout: 0x00000000
			Pairable: no
			UUID: A/V Remote Control        (0000110e-0000-1000-8000-00805f9b34fb)
			UUID: Audio Source              (0000110a-0000-1000-8000-00805f9b34fb)
			UUID: PnP Information           (00001200-0000-1000-8000-00805f9b34fb)
			UUID: Audio Sink                (0000110b-0000-1000-8000-00805f9b34fb)
			UUID: A/V Remote Control Target (0000110c-0000-1000-8000-00805f9b34fb)
			UUID: Generic Access Profile    (00001800-0000-1000-8000-00805f9b34fb)
			UUID: Generic Attribute Profile (00001801-0000-1000-8000-00805f9b34fb)
			UUID: Device Information        (0000180a-0000-1000-8000-00805f9b34fb)
			Modalias: usb:v1D6Bp0246d0537
			Discovering: no
			Roles: central
			Roles: peripheral
		Advertising Features:
			ActiveInstances: 0x00 (0)
			SupportedInstances: 0x05 (5)
			SupportedIncludes: tx-power
			SupportedIncludes: appearance
			SupportedIncludes: local-name
	*/

	return nil
}

func (bt *Service) getBluetoothConnectionCount(device string) (int, error) {
	cmd := bt.CmdExecutor.Command("bluetoothctl", "info", device)
	err := cmd.Run()
	if err != nil {
		return 0, err
	}

	return bt.countConnectedLines(bt.CmdExecutor.GetOutput()), nil
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
