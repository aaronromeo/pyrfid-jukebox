package bluetooth

import (
	"context"
	"os/exec"
	"time"
)

const commandTimeout = 15 * time.Second

// Runner is the single impure seam for external commands.
type Runner interface {
	Output(name string, args ...string) (string, error)
	Run(name string, args ...string) error
}

type OSRunner struct{}

func (OSRunner) Output(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return string(out), err
}

func (OSRunner) Run(name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Run()
}
