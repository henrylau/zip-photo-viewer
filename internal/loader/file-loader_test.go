package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func createTestImageFile(dir, name string) string {
	path := filepath.Join(dir, name)
	os.WriteFile(path, []byte("fake image data"), 0644)
	return path
}

func TestFileLoader_LoadAndNavigation(t *testing.T) {
	// Setup: create temp dir and files
	tmpDir := t.TempDir()
	createTestImageFile(tmpDir, "a.jpg")
	createTestImageFile(tmpDir, "b.png")

	// Create loader and load folder
	loader := NewFileLoader()
	err := loader.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loader.TotalImage() != 2 {
		t.Errorf("Expected 2 images, got %d", loader.TotalImage())
	}

	// Test Get
	data, err := loader.Get()
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Get returned empty data")
	}

	// Test Next
	data, err = loader.Next()
	if err != nil {
		t.Errorf("Next failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Next returned empty data")
	}

	// Test Prev
	data, err = loader.Prev()
	if err != nil {
		t.Errorf("Prev failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Prev returned empty data")
	}

	// Test GetInfo
	info := loader.GetInfo()
	if info.Name == "" || info.FilePath == "" || info.Size == 0 || info.ModTime.IsZero() {
		t.Error("GetInfo returned incomplete info")
	}

	// Test Index
	if loader.Index() < 0 || loader.Index() > 1 {
		t.Errorf("Index out of range: %d", loader.Index())
	}

	// Test Close
	if err := loader.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
