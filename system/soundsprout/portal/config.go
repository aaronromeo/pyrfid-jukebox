package portal

import (
	"fmt"

	"aaronromeo.com/soundsprout/bluetooth"
	"aaronromeo.com/soundsprout/templategen"
)

const sudoCommand = "sudo"

type AudioConfig struct {
	asoundrcPath string
	cmusRcPath   string
	runner       bluetooth.Runner
}

// NewAudioConfig returns an AudioConfig that renders and applies audio configs.
func NewAudioConfig(asoundrcPath, cmusRcPath string, runner bluetooth.Runner) *AudioConfig {
	return &AudioConfig{
		asoundrcPath: asoundrcPath,
		cmusRcPath:   cmusRcPath,
		runner:       runner,
	}
}

// Apply renders the templated audio configs for mac, writes them, and restarts
// the services that consume them.
func (a *AudioConfig) Apply(mac string) error {
	subs := map[string]string{"PJ_BLUETOOTH_DEVICE": mac}

	asoundrc, err := templategen.GenerateTemplate("templates/asoundrc.txt", subs)
	if err != nil {
		return fmt.Errorf("render asoundrc: %w", err)
	}
	cmusrc, err := templategen.GenerateTemplate("templates/config-cmus-rc.txt", subs)
	if err != nil {
		return fmt.Errorf("render cmus rc: %w", err)
	}
	if err := writeFileAtomic(a.asoundrcPath, asoundrc); err != nil {
		return err
	}
	if err := writeFileAtomic(a.cmusRcPath, cmusrc); err != nil {
		return err
	}

	commands := [][]string{
		{sudoCommand, "supervisorctl", "restart", "btconnect"},
		{sudoCommand, "supervisorctl", "restart", "cmus_manager"},
		{sudoCommand, "alsactl", "restore"},
	}
	for _, c := range commands {
		if err := a.runner.Run(c[0], c[1:]...); err != nil {
			return fmt.Errorf("run %v: %w", c, err)
		}
	}
	return nil
}
