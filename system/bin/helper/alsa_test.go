package helper_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	helper "aaronromeo.com/rfid-jukebox/system/bin/helper"
)

type MockCmd struct {
	Executed bool
}

func (c *MockCmd) Run() error {
	c.Executed = true

	return nil
}

type MockCommandExecutor struct {
	CommandExecuted bool
	Output          string
}

func (e *MockCommandExecutor) Command(_ string, _ ...string) helper.Cmd {
	e.CommandExecuted = true

	return &MockCmd{}
}

func (e *MockCommandExecutor) GetOutput() string {
	return ""
}

func TestFilesAreDifferent(t *testing.T) {
	// Test 1: Two identical files
	err := os.WriteFile("test1.txt", []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile("test2.txt", []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := helper.FilesAreDifferent("test1.txt", "test2.txt")
	if err != nil {
		t.Fatal(err)
	}
	if diff {
		t.Errorf("Expected files to be identical, but they were different")
	}

	// Test 2: Two different files
	err = os.WriteFile("test2.txt", []byte("test2"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	diff, err = helper.FilesAreDifferent("test1.txt", "test2.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !diff {
		t.Errorf("Expected files to be different, but they were identical")
	}

	// Test 3: File does not exist
	_, err = helper.FilesAreDifferent("test1.txt", "nonexistent.txt")
	if err == nil {
		t.Errorf("Expected error, but got none")
	}

	// Cleanup
	os.Remove("test1.txt")
	os.Remove("test2.txt")
}

func TestHasALSAConfigChanged(t *testing.T) {
	// Setup test files
	testDir, err := os.MkdirTemp(".", "test_data")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)
	err = os.MkdirAll(filepath.Join(testDir, "system", "home"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	testSystemConfig := filepath.Join(testDir, "system.asoundrc")
	testRepoConfig := filepath.Join(filepath.Join(testDir, "system", "home"), ".asoundrc")

	t.Setenv("PJ_ALSA_CONFIG", testSystemConfig)
	t.Setenv("PJ_PROJECT_ROOT", testDir)

	// Test 1: Identical Files
	err = os.WriteFile(testSystemConfig, []byte("test config"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(testRepoConfig, []byte("test config"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := helper.HasALSAConfigChanged()
	if err != nil {
		t.Fatal(err)
	}
	if diff {
		t.Errorf("Expected ALSA configs to be identical, but they were reported as different")
	}

	// Test 2: Different Files
	err = os.WriteFile(testRepoConfig, []byte("different config"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	diff, err = helper.HasALSAConfigChanged()
	if err != nil {
		t.Fatal(err)
	}
	if !diff {
		t.Errorf("Expected ALSA configs to be different, but they were reported as identical")
	}

	// Test 3: File Does Not Exist
	os.Remove(testRepoConfig)
	_, err = helper.HasALSAConfigChanged()
	if err == nil {
		t.Errorf("Expected error for missing file, but got none")
	}

	// Optionally, add more tests for scenarios with environment variables set
}
func TestUpdateALSAConfig(t *testing.T) {
	// Setup test files
	testDir, err := os.MkdirTemp(".", "test_data")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)
	err = os.MkdirAll(filepath.Join(testDir, "system", "home"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	alsaConfigUpdate := helper.RealALSAConfigUpdater{}

	testSystemConfig := filepath.Join(testDir, "system.asoundrc")
	testRepoConfig := filepath.Join(filepath.Join(testDir, "system", "home"), ".asoundrc")

	t.Setenv("PJ_ALSA_CONFIG", testSystemConfig)
	t.Setenv("PJ_PROJECT_ROOT", testDir)

	// Test 1: ALSA config has not changed
	err = os.WriteFile(testSystemConfig, []byte("test config"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(testRepoConfig, []byte("test config"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmdExecutor := &MockCommandExecutor{}
	err = alsaConfigUpdate.UpdateALSAConfig(cmdExecutor)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if cmdExecutor.CommandExecuted {
		t.Errorf("Unexpected command execution")
	}

	// Test 2: ALSA config has changed
	err = os.WriteFile(testRepoConfig, []byte("different config"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmdExecutor = &MockCommandExecutor{}
	err = alsaConfigUpdate.UpdateALSAConfig(cmdExecutor)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !cmdExecutor.CommandExecuted {
		t.Errorf("Expected command execution")
	}
}

func TestCopyFile(t *testing.T) {
	src := "test_src.txt"
	dst := "test_dst.txt"

	// Test 1: Successful copy
	// Create a test source file
	err := os.WriteFile(src, []byte("test content"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(src)

	// Call the CopyFile function
	err = helper.CopyFile(src, dst)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify that the destination file exists and has the same content as the source file
	dstContent, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}

	srcContent, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}

	if string(dstContent) != string(srcContent) {
		t.Errorf("Expected destination file content to be the same as source file content")
	}

	// Cleanup
	os.Remove(dst)
}
func TestIsALSARunning(t *testing.T) {
	cmdExecutor := &MockCommandExecutor{}
	alsaConfigUpdater := helper.RealALSAConfigUpdater{}

	// Test 1: ALSA is running
	cmdExecutor.CommandExecuted = false
	cmdExecutor.Output = "bluealsa.service - BluezALSA proxy\n   Loaded: loaded (/lib/systemd/system/bluealsa.service; enabled; vendor preset: enabled)\n   Active: active (running) since Wed 2022-10-12 10:00:00 UTC; 1 day 10h ago\n     Docs: man:bluealsa(1)\n Main PID: 1234 (bluealsa)\n    Tasks: 1 (limit: 4915)\n   Memory: 1.2M\n   CGroup: /system.slice/bluealsa.service\n           └─1234 /usr/bin/bluealsa --profile=a2dp-sink\n"
	expectedResult := true
	expectedError := error(nil)

	result, err := alsaConfigUpdater.IsALSARunning(cmdExecutor)

	if result != expectedResult {
		t.Errorf("Expected ALSA to be running, but it is not")
	}
	if err != expectedError {
		t.Errorf("Expected no error, but got: %v", err)
	}
	if !cmdExecutor.CommandExecuted {
		t.Errorf("Expected command execution")
	}

	// Test 2: ALSA is not running
	cmdExecutor.CommandExecuted = false
	cmdExecutor.Output = "bluealsa.service - BluezALSA proxy\n   Loaded: loaded (/lib/systemd/system/bluealsa.service; enabled; vendor preset: enabled)\n   Active: inactive (dead) since Wed 2022-10-12 10:00:00 UTC; 1 day 10h ago\n     Docs: man:bluealsa(1)\n Main PID: 1234 (code=exited, status=0/SUCCESS)\n    Tasks: 0 (limit: 4915)\n   Memory: 0B\n   CGroup: /system.slice/bluealsa.service\n"
	expectedResult = false
	expectedError = error(nil)

	result, err = alsaConfigUpdater.IsALSARunning(cmdExecutor)

	if result != expectedResult {
		t.Errorf("Expected ALSA to not be running, but it is")
	}
	if err != expectedError {
		t.Errorf("Expected no error, but got: %v", err)
	}
	if !cmdExecutor.CommandExecuted {
		t.Errorf("Expected command execution")
	}

	// Test 3: Error executing command
	cmdExecutor.CommandExecuted = false
	cmdExecutor.Output = ""
	expectedResult = false
	expectedError = fmt.Errorf("command execution error")

	result, err = alsaConfigUpdater.IsALSARunning(cmdExecutor)

	if result != expectedResult {
		t.Errorf("Expected ALSA to not be running, but it is")
	}
	if err == nil || err.Error() != expectedError.Error() {
		t.Errorf("Expected error: %v, but got: %v", expectedError, err)
	}
	if !cmdExecutor.CommandExecuted {
		t.Errorf("Expected command execution")
	}
}
