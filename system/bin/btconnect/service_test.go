package btconnect_test

import (
	"bytes"
	"errors"
	"log"
	"os"
	"testing"

	btconnect "aaronromeo.com/rfid-jukebox/system/bin/btconnect"
	helper "aaronromeo.com/rfid-jukebox/system/bin/helper"
)

type MockALSAConfigUpdater struct {
	UpdateALSAConfigFunc func(cmdExecutor helper.CommandExecutor) error
	IsALSARunningFunc    func(cmdExecutor helper.CommandExecutor) (bool, error)
}

func (m *MockALSAConfigUpdater) UpdateALSAConfig(cmdExecutor helper.CommandExecutor) error {
	if m.UpdateALSAConfigFunc != nil {
		return m.UpdateALSAConfigFunc(cmdExecutor)
	}
	return nil
}

func (m *MockALSAConfigUpdater) IsALSARunning(cmdExecutor helper.CommandExecutor) (bool, error) {
	if m.IsALSARunningFunc != nil {
		return m.IsALSARunningFunc(cmdExecutor)
	}
	return false, nil
}

type MockCmd struct {
	Output  string
	IsError bool
}

func (m *MockCmd) Run() error {
	if m.IsError {
		return errors.New(m.Output)
	}
	return nil
}

type MockCmdExecutor struct {
	CommandExecuted bool
	Output          string
}

func (m *MockCmdExecutor) Command(name string, arg ...string) helper.Cmd {
	m.CommandExecuted = true

	var isError bool

	switch {
	case name == "bluetoothctl" && arg[0] == "info" && arg[1] == "connected_device":
		isError = false
		m.Output = `
			Device 88:C6:26:23:95:3F (public)
			Name: UE MINI BOOM
			Alias: UE MINI BOOM
			Class: 0x00240404
			Icon: audio-card
			Paired: yes
			Trusted: yes
			Blocked: no
			Connected: yes
			LegacyPairing: no
			UUID: Vendor specific           (00000000-deca-fade-deca-deafdecacaff)
			UUID: Serial Port               (00001101-0000-1000-8000-00805f9b34fb)
			UUID: Audio Sink                (0000110b-0000-1000-8000-00805f9b34fb)
			UUID: A/V Remote Control Target (0000110c-0000-1000-8000-00805f9b34fb)
			UUID: Advanced Audio Distribu.. (0000110d-0000-1000-8000-00805f9b34fb)
			UUID: A/V Remote Control        (0000110e-0000-1000-8000-00805f9b34fb)
			UUID: Handsfree                 (0000111e-0000-1000-8000-00805f9b34fb)
			UUID: PnP Information           (00001200-0000-1000-8000-00805f9b34fb)
			Modalias: usb:v046DpBA20dFF0E
			`
	case name == "bluetoothctl" && arg[0] == "info" && arg[1] == "not_connected_device":
		isError = false
		m.Output = `
			Device FC:58:FA:8C:E3:A8 (public)
			Name: ENEBY20
			Alias: ENEBY20
			Class: 0x002c0418
			Icon: audio-card
			Paired: yes
			Trusted: yes
			Blocked: no
			Connected: no
			LegacyPairing: no
			UUID: Serial Port               (00001101-0000-1000-8000-00805f9b34fb)
			UUID: Audio Sink                (0000110b-0000-1000-8000-00805f9b34fb)
			UUID: A/V Remote Control Target (0000110c-0000-1000-8000-00805f9b34fb)
			UUID: Advanced Audio Distribu.. (0000110d-0000-1000-8000-00805f9b34fb)
			UUID: A/V Remote Control        (0000110e-0000-1000-8000-00805f9b34fb)
			UUID: PnP Information           (00001200-0000-1000-8000-00805f9b34fb)
			Modalias: bluetooth:v000ApFFFFdFFFF
			`
	case name == "bluetoothctl" && arg[0] == "info" && arg[1] == "non_existant_device":
		isError = true
		m.Output = `
			Device bacon not available
			`
	}
	return &MockCmd{Output: m.Output, IsError: isError}
}

func (m *MockCmdExecutor) GetOutput() string {
	return m.Output
}

func TestService_Run(t *testing.T) {
	// Setup
	var device string
	var buf bytes.Buffer
	log.SetOutput(&buf)

	// Create a mock ALSAConfigUpdater
	mockALSAConfigUpdater := &MockALSAConfigUpdater{}

	// Create a mock CmdExecutor
	mockCmdExecutor := &MockCmdExecutor{}

	// Create a new Service instance
	btService := *btconnect.NewBtConnectService(mockCmdExecutor, mockALSAConfigUpdater)

	// Test 1: ALSA config update succeeds and Bluetooth connection count is retrieved successfully
	device = "connected_device"
	t.Setenv("PJ_BLUETOOTH_DEVICE", device)

	mockALSAConfigUpdater.UpdateALSAConfigFunc = func(cmdExecutor helper.CommandExecutor) error {
		return nil
	}

	mockCmdExecutor.CommandExecuted = false

	err := btService.Run()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check the log output
	expectedLog := "Number of connections: 1"
	if !bytes.Contains(buf.Bytes(), []byte(expectedLog)) {
		t.Errorf("Expected log output '%s', got '%s'", expectedLog, buf.String())
	}

	// Test 2: ALSA config update fails
	device = "connected_device"
	t.Setenv("PJ_BLUETOOTH_DEVICE", device)
	mockALSAConfigUpdater.UpdateALSAConfigFunc = func(cmdExecutor helper.CommandExecutor) error {
		return errors.New("ALSA config update failed")
	}

	err = btService.Run()
	if err == nil {
		t.Errorf("Expected error, but got none")
	}

	// Test 3: Bluetooth connection count retrieval fails
	device = "non_existant_device"
	t.Setenv("PJ_BLUETOOTH_DEVICE", device)
	mockALSAConfigUpdater.UpdateALSAConfigFunc = func(cmdExecutor helper.CommandExecutor) error {
		return nil
	}

	mockCmdExecutor.CommandExecuted = false

	err = btService.Run()
	if err == nil {
		t.Errorf("Expected error, but got none")
	}

	// Test 4: PJ_BLUETOOTH_DEVICE is not set
	os.Unsetenv("PJ_BLUETOOTH_DEVICE")
	mockALSAConfigUpdater.UpdateALSAConfigFunc = func(cmdExecutor helper.CommandExecutor) error {
		return nil
	}

	mockCmdExecutor.CommandExecuted = false

	err = btService.Run()
	if err == nil {
		t.Errorf("Expected error, but got none")
	}

	// Test 5: Bluetooth connection count is 0
	device = "not_connected_device"
	t.Setenv("PJ_BLUETOOTH_DEVICE", device)

	mockALSAConfigUpdater.UpdateALSAConfigFunc = func(cmdExecutor helper.CommandExecutor) error {
		return nil
	}

	mockCmdExecutor.CommandExecuted = false

	err = btService.Run()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check the log output
	expectedLog = "Number of connections: 0"
	// TODO: This test needs to be revisited. A count of 0 should result in some corrective action.
	if !bytes.Contains(buf.Bytes(), []byte(expectedLog)) {
		t.Errorf("Expected log output '%s', got '%s'", expectedLog, buf.String())
	}

	// Cleanup
	os.Unsetenv("PJ_BLUETOOTH_DEVICE")
}
