package bluetooth

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	bluetoothctlCommand = "bluetoothctl"
	yesValue            = "yes"
)

// ValidMAC reports whether mac is a colon-separated 48-bit address.
func ValidMAC(mac string) bool {
	re := regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)
	return re.MatchString(mac)
}

type Device struct {
	MAC  string
	Name string
}

type Info struct {
	MAC       string
	Name      string
	Paired    bool
	Trusted   bool
	Connected bool
}

// ParseInfo parses the output of `bluetoothctl info <mac>`.
func ParseInfo(out string) Info {
	info := Info{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Device "):
			if fields := strings.Fields(line); len(fields) >= 2 {
				info.MAC = fields[1]
			}
		case strings.HasPrefix(line, "Name:"):
			info.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		case strings.HasPrefix(line, "Paired:"):
			info.Paired = strings.Contains(line, yesValue)
		case strings.HasPrefix(line, "Trusted:"):
			info.Trusted = strings.Contains(line, yesValue)
		case strings.HasPrefix(line, "Connected:"):
			info.Connected = strings.Contains(line, yesValue)
		}
	}
	return info
}

// ParseDevices parses `bluetoothctl devices` lines, sorted by MAC.
func ParseDevices(out string) []Device {
	var devices []Device
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && fields[0] == "Device" {
			name := ""
			if len(fields) > 2 {
				name = strings.Join(fields[2:], " ")
			}
			devices = append(devices, Device{MAC: fields[1], Name: name})
		}
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].MAC < devices[j].MAC })
	return devices
}

// Status returns the parsed info for mac.
func Status(r Runner, mac string) (Info, error) {
	out, err := r.Output(bluetoothctlCommand, "info", mac)
	if err != nil {
		return Info{}, fmt.Errorf("bluetoothctl info %s: %w", mac, err)
	}
	return ParseInfo(out), nil
}

// IsConnected reports whether mac is currently connected.
func IsConnected(r Runner, mac string) (bool, error) {
	info, err := Status(r, mac)
	if err != nil {
		return false, err
	}
	return info.Connected, nil
}

// Connect connects to an already-paired mac.
func Connect(r Runner, mac string) error {
	if err := r.Run(bluetoothctlCommand, "connect", mac); err != nil {
		return fmt.Errorf("bluetoothctl connect %s: %w", mac, err)
	}
	return nil
}

// Reconnect connects to mac only if it is not already connected.
func Reconnect(r Runner, mac string) error {
	connected, err := IsConnected(r, mac)
	if err != nil {
		return err
	}
	if connected {
		return nil
	}
	return Connect(r, mac)
}

// KnownDevices lists the devices bluetoothctl already knows about.
func KnownDevices(r Runner) ([]Device, error) {
	out, err := r.Output(bluetoothctlCommand, "devices")
	if err != nil {
		return nil, fmt.Errorf("bluetoothctl devices: %w", err)
	}
	return ParseDevices(out), nil
}

// Scan runs a timed scan, then returns the known devices.
func Scan(r Runner, timeout time.Duration) ([]Device, error) {
	secs := int(timeout.Seconds())
	if secs < 1 {
		secs = 1
	}
	if _, err := r.Output(bluetoothctlCommand, "--timeout", fmt.Sprintf("%d", secs), "scan", "on"); err != nil {
		return nil, fmt.Errorf("bluetoothctl scan: %w", err)
	}
	return KnownDevices(r)
}

// Pair pairs, trusts, and connects mac.
func Pair(r Runner, mac string) error {
	if !ValidMAC(mac) {
		return fmt.Errorf("invalid MAC address %q", mac)
	}
	if err := r.Run(bluetoothctlCommand, "pair", mac); err != nil {
		return fmt.Errorf("pair %s: %w", mac, err)
	}
	if err := r.Run(bluetoothctlCommand, "trust", mac); err != nil {
		return fmt.Errorf("trust %s: %w", mac, err)
	}
	return Connect(r, mac)
}
