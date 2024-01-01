package btconnect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilesAreDifferent(t *testing.T) {
	service := &Service{}

	// Test 1: Two identical files
	err := os.WriteFile("test1.txt", []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile("test2.txt", []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := service.filesAreDifferent("test1.txt", "test2.txt")
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
	diff, err = service.filesAreDifferent("test1.txt", "test2.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !diff {
		t.Errorf("Expected files to be different, but they were identical")
	}

	// Test 3: File does not exist
	_, err = service.filesAreDifferent("test1.txt", "nonexistent.txt")
	if err == nil {
		t.Errorf("Expected error, but got none")
	}

	// Cleanup
	os.Remove("test1.txt")
	os.Remove("test2.txt")
}

func TestHasALSAConfigChanged(t *testing.T) {
	service := &Service{}

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

	os.Setenv("PJ_ALSA_CONFIG", testSystemConfig)
	os.Setenv("PJ_PROJECT_ROOT", testDir)

	// Test 1: Identical Files
	err = os.WriteFile(testSystemConfig, []byte("test config"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(testRepoConfig, []byte("test config"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := service.hasALSAConfigChanged()
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
	diff, err = service.hasALSAConfigChanged()
	if err != nil {
		t.Fatal(err)
	}
	if !diff {
		t.Errorf("Expected ALSA configs to be different, but they were reported as identical")
	}

	// Test 3: File Does Not Exist
	os.Remove(testRepoConfig)
	_, err = service.hasALSAConfigChanged()
	if err == nil {
		t.Errorf("Expected error for missing file, but got none")
	}

	// Optionally, add more tests for scenarios with environment variables set
}
