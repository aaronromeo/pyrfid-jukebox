package helper

import (
	"bytes"
	"os/exec"
)

type CommandExecutor interface {
	Command(name string, arg ...string) Cmd
	GetOutput() string
}

type Cmd interface {
	Run() error
}

type OSCommandExecutor struct {
	out bytes.Buffer
}

func (e *OSCommandExecutor) Command(name string, arg ...string) Cmd {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = &e.out

	return cmd
}

func (e *OSCommandExecutor) GetOutput() string {
	return e.out.String()
}

type ALSAConfigUpdater interface {
	UpdateALSAConfig(executor CommandExecutor) error
	IsALSARunning(executor CommandExecutor) (bool, error)
}

type RealALSAConfigUpdater struct{}
