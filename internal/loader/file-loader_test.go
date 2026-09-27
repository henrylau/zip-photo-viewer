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

func TestNewLoader_Folder(t *testing.T) {
	tmpDir := t.TempDir()
	createTestImageFile(tmpDir, "a.jpg")
	createTestImageFile(tmpDir, "b.png")

	l, err := NewLoader(tmpDir)
	if err != nil {
		t.Fatalf("NewLoader(folder) failed: %v", err)
	}
	if l.TotalImage() != 2 {
		t.Errorf("Expected 2 images, got %d", l.TotalImage())
	}
	data, err := l.Get()
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Get returned empty data")
	}
	if err := l.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestFileLoader_EntriesAndSeek(t *testing.T) {
	tmpDir := t.TempDir()
	createTestImageFile(tmpDir, "a.jpg")
	createTestImageFile(tmpDir, "b.png")

	loader := NewFileLoader()
	if err := loader.Load(tmpDir); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	entries := loader.Entries()
	if len(entries) != 2 {
		t.Fatalf("Entries() = %d, want 2", len(entries))
	}

	target := 1
	if entries[0].Name != "a.jpg" {
		target = 0
	}
	wantName := entries[target].Name

	if _, err := loader.Seek(target); err != nil {
		t.Fatalf("Seek failed: %v", err)
	}
	if loader.Index() != target {
		t.Errorf("Index() = %d, want %d", loader.Index(), target)
	}
	if loader.GetInfo().Name != wantName {
		t.Errorf("GetInfo().Name = %s, want %s", loader.GetInfo().Name, wantName)
	}
	if _, err := loader.Seek(-1); err == nil {
		t.Error("Seek(-1) should fail")
	}
}

func TestFileLoader_ChildFoldersAndOpenFolder(t *testing.T) {
	tmpDir := t.TempDir()
	aDir := filepath.Join(tmpDir, "a")
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.Mkdir(aDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestImageFile(tmpDir, "root.jpg")
	createTestImageFile(aDir, "in-a.jpg")

	loader := NewFileLoader()
	if err := loader.Load(tmpDir); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	folders := loader.ChildFolders()
	var names []string
	for _, f := range folders {
		names = append(names, f.Name)
	}
	foundA := false
	for _, name := range names {
		if name == "empty" {
			t.Fatalf("ChildFolders included empty dir: %v", names)
		}
		if name == "a" {
			foundA = true
		}
	}
	if !foundA {
		t.Fatalf("ChildFolders = %v, want a", names)
	}

	if err := loader.OpenFolder(aDir); err != nil {
		t.Fatalf("OpenFolder failed: %v", err)
	}
	entries := loader.Entries()
	if len(entries) != 1 || entries[0].Name != "in-a.jpg" {
		t.Fatalf("after OpenFolder entries = %+v", entries)
	}

	folders = loader.ChildFolders()
	if len(folders) == 0 || folders[0].Name != ".." {
		t.Fatalf("ChildFolders from a/ = %+v, want .. first", folders)
	}
}

func TestFileLoader_ChildArchives(t *testing.T) {
	tmpDir := t.TempDir()
	boxed := filepath.Join(tmpDir, "boxed")
	if err := os.Mkdir(boxed, 0755); err != nil {
		t.Fatal(err)
	}
	createTestImageFile(tmpDir, "a.jpg")
	os.WriteFile(filepath.Join(tmpDir, "album.zip"), []byte("zip"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("txt"), 0644)
	os.WriteFile(filepath.Join(boxed, "pics.7z"), []byte("7z"), 0644)

	loader := NewFileLoader()
	if err := loader.Load(tmpDir); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	archives := loader.ChildArchives()
	if len(archives) != 1 || archives[0].Name != "album.zip" {
		t.Fatalf("ChildArchives = %+v, want album.zip", archives)
	}

	var folderNames []string
	for _, f := range loader.ChildFolders() {
		folderNames = append(folderNames, f.Name)
	}
	foundBoxed := false
	for _, name := range folderNames {
		if name == "boxed" {
			foundBoxed = true
		}
	}
	if !foundBoxed {
		t.Fatalf("ChildFolders = %v, want boxed (archive-only folder)", folderNames)
	}
}
