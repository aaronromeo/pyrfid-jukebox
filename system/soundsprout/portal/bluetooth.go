package portal

import (
	"fmt"
	"time"

	"aaronromeo.com/soundsprout/bluetooth"
)

const bluetoothDeviceKey = "PJ_BLUETOOTH_DEVICE"

type Bluetooth struct {
	runner   bluetooth.Runner
	settings *Settings
	audio    *AudioConfig
}

// NewBluetooth returns a façade over bluetoothctl plus config apply.
func NewBluetooth(runner bluetooth.Runner, settings *Settings, audio *AudioConfig) *Bluetooth {
	return &Bluetooth{runner: runner, settings: settings, audio: audio}
}

// ActiveMAC returns the configured device MAC, or "" if none.
func (b *Bluetooth) ActiveMAC() (string, error) {
	return b.settings.Get(bluetoothDeviceKey)
}

// Status returns the parsed status for mac.
func (b *Bluetooth) Status(mac string) (bluetooth.Info, error) {
	return bluetooth.Status(b.runner, mac)
}

// KnownDevices lists devices bluetoothctl already knows about.
func (b *Bluetooth) KnownDevices() ([]bluetooth.Device, error) {
	return bluetooth.KnownDevices(b.runner)
}

// Scan runs a timed scan and returns the known devices.
func (b *Bluetooth) Scan(timeout time.Duration) ([]bluetooth.Device, error) {
	return bluetooth.Scan(b.runner, timeout)
}

// Activate makes an already-paired device the active audio output.
func (b *Bluetooth) Activate(mac string) error {
	if !bluetooth.ValidMAC(mac) {
		return fmt.Errorf("invalid MAC address %q", mac)
	}
	if err := bluetooth.Connect(b.runner, mac); err != nil {
		return err
	}
	return b.persistAndApply(mac)
}

// PairAndActivate pairs, trusts, and connects a new device, then applies it.
func (b *Bluetooth) PairAndActivate(mac string) error {
	if !bluetooth.ValidMAC(mac) {
		return fmt.Errorf("invalid MAC address %q", mac)
	}
	if err := bluetooth.Pair(b.runner, mac); err != nil {
		return err
	}
	return b.persistAndApply(mac)
}

// Reconnect reconnects the currently configured device.
func (b *Bluetooth) Reconnect() error {
	mac, err := b.ActiveMAC()
	if err != nil {
		return err
	}
	if mac == "" {
		return fmt.Errorf("no active bluetooth device configured")
	}
	return bluetooth.Reconnect(b.runner, mac)
}

func (b *Bluetooth) persistAndApply(mac string) error {
	if err := b.settings.Set(bluetoothDeviceKey, mac); err != nil {
		return err
	}
	return b.audio.Apply(mac)
}
