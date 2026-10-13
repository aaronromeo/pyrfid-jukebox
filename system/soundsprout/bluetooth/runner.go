package bluetooth

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

const (
	commandTimeout        = 15 * time.Second
	connectCommandTimeout = 45 * time.Second
)

// OSRunner executes external commands under a deadline. The zero value uses
// the package defaults; tests inject short timeouts.
type OSRunner struct {
	// CommandTimeout bounds quick commands such as `bluetoothctl info`.
	CommandTimeout time.Duration
	// ConnectTimeout bounds blocking commands whose first argument is
	// `connect` or `pair`, which can take tens of seconds on an
	// unreachable device.
	ConnectTimeout time.Duration
}

// Runner is the single impure seam for external commands.
type Runner interface {
	Output(name string, args ...string) (string, error)
	Run(name string, args ...string) error
}

func (r OSRunner) timeoutFor(name string, args []string) time.Duration {
	if name == bluetoothctlCommand && len(args) > 0 &&
		(args[0] == "connect" || args[0] == "pair") {
		if r.ConnectTimeout > 0 {
			return r.ConnectTimeout
		}
		return connectCommandTimeout
	}
	if r.CommandTimeout > 0 {
		return r.CommandTimeout
	}
	return commandTimeout
}

func (r OSRunner) commandContext(name string, args []string) (context.Context, context.CancelFunc, time.Duration) {
	d := r.timeoutFor(name, args)
	ctx, cancel := context.WithTimeout(context.Background(), d)
	return ctx, cancel, d
}

func (r OSRunner) Output(name string, args ...string) (string, error) {
	ctx, cancel, d := r.commandContext(name, args)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", fmt.Errorf("timed out after %s", d)
	}
	return string(out), err
}

func (r OSRunner) Run(name string, args ...string) error {
	ctx, cancel, d := r.commandContext(name, args)
	defer cancel()
	err := exec.CommandContext(ctx, name, args...).Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("timed out after %s", d)
	}
	return err
}
