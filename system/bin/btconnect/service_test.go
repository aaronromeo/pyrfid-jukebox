package btconnect

import (
	"os"
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
