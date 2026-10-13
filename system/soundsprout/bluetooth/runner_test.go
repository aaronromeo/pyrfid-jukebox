package bluetooth_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"aaronromeo.com/soundsprout/bluetooth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCommandTimeout = 200 * time.Millisecond

func TestOSRunnerCommandTimeoutKillsSlowCommands(t *testing.T) {
	r := bluetooth.OSRunner{CommandTimeout: testCommandTimeout}

	err := r.Run("sleep", "1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out after")
}

func TestOSRunnerConnectTimeoutUsesLongDeadline(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "bluetoothctl")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nsleep 5\n"), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	r := bluetooth.OSRunner{ConnectTimeout: testCommandTimeout}

	err := r.Run("bluetoothctl", "connect", "AA:BB:CC:DD:EE:FF")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out after")
}

func TestOSRunnerQuickCommandWithinTimeoutSucceeds(t *testing.T) {
	r := bluetooth.OSRunner{CommandTimeout: 10 * time.Second}

	require.NoError(t, r.Run("sleep", "1"))
}
