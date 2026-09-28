package loader

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeZipArchive(t *testing.T, dir string, files map[string][]byte) string {
	t.Helper()
	path := filepath.Join(dir, "photos.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return path
}

func writeTarGzArchive(t *testing.T, dir string, files map[string][]byte) string {
	t.Helper()
	path := filepath.Join(dir, "photos.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create tar.gz: %v", err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, data := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(data)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header %s: %v", name, err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatalf("write tar entry %s: %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return path
}

func assertArchiveNavigation(t *testing.T, path string) {
	t.Helper()
	loader := NewArchiveLoader()
	if err := loader.Load(path); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loader.TotalImage() != 2 {
		t.Errorf("Expected 2 images, got %d", loader.TotalImage())
	}

	data, err := loader.Get()
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Get returned empty data")
	}
	first := loader.GetInfo().Name

	data, err = loader.Next()
	if err != nil {
		t.Errorf("Next failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Next returned empty data")
	}
	if loader.GetInfo().Name == first {
		t.Error("Next did not advance to a different image")
	}

	data, err = loader.Prev()
	if err != nil {
		t.Errorf("Prev failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Prev returned empty data")
	}
	if loader.GetInfo().Name != first {
		t.Errorf("Prev did not return to first image, got %s", loader.GetInfo().Name)
	}

	info := loader.GetInfo()
	if info.Name == "" || info.FilePath == "" || info.Size == 0 {
		t.Error("GetInfo returned incomplete info")
	}

	if loader.Index() < 0 || loader.Index() > 1 {
		t.Errorf("Index out of range: %d", loader.Index())
	}

	if err := loader.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestArchiveLoader_NoMedia(t *testing.T) {
	tmpDir := t.TempDir()
	path := writeZipArchive(t, tmpDir, map[string][]byte{
		"notes.txt": []byte("hello"),
		"readme.md": []byte("docs"),
	})
	loader := NewArchiveLoader()
	if err := loader.Load(path); !errors.Is(err, ErrNoMedia) {
		t.Fatalf("Load() error = %v, want ErrNoMedia", err)
	}

	emptyPath := writeZipArchive(t, t.TempDir(), map[string][]byte{})
	if err := loader.Load(emptyPath); !errors.Is(err, ErrNoMedia) {
		t.Fatalf("empty zip Load() error = %v, want ErrNoMedia", err)
	}

	if _, err := NewLoader(path); !errors.Is(err, ErrNoMedia) {
		t.Fatalf("NewLoader() error = %v, want ErrNoMedia", err)
	}
}

func TestArchiveLoader_Zip(t *testing.T) {
	tmpDir := t.TempDir()
	path := writeZipArchive(t, tmpDir, map[string][]byte{
		"a.jpg": []byte("fake jpeg a"),
		"b.png": []byte("fake png b"),
	})
	assertArchiveNavigation(t, path)
}

func TestArchiveLoader_Seek(t *testing.T) {
	tmpDir := t.TempDir()
	path := writeZipArchive(t, tmpDir, map[string][]byte{
		"a.jpg": []byte("fake jpeg a"),
		"b.png": []byte("fake png b"),
	})
	loader := NewArchiveLoader()
	if err := loader.Load(path); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	entries := loader.Entries()
	if len(entries) != 2 {
		t.Fatalf("Entries() = %d, want 2", len(entries))
	}
	if _, err := loader.Seek(1); err != nil {
		t.Fatalf("Seek failed: %v", err)
	}
	if loader.Index() != 1 {
		t.Errorf("Index() = %d, want 1", loader.Index())
	}
	if loader.GetInfo().Name != entries[1].Name {
		t.Errorf("GetInfo().Name = %s, want %s", loader.GetInfo().Name, entries[1].Name)
	}
	if _, err := loader.Seek(99); err == nil {
		t.Error("Seek(99) should fail")
	}
}

func TestArchiveLoader_ZipWithDummyPassword(t *testing.T) {
	tmpDir := t.TempDir()
	path := writeZipArchive(t, tmpDir, map[string][]byte{
		"a.jpg": []byte("fake jpeg a"),
		"b.png": []byte("fake png b"),
	})
	loader := NewArchiveLoader()
	if err := loader.LoadWithPassword(path, "unused"); err != nil {
		t.Fatalf("LoadWithPassword failed: %v", err)
	}
	if loader.TotalImage() != 2 {
		t.Errorf("Expected 2 images, got %d", loader.TotalImage())
	}
	if _, err := loader.Get(); err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if err := loader.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestNewLoaderWithPassword_Zip(t *testing.T) {
	tmpDir := t.TempDir()
	path := writeZipArchive(t, tmpDir, map[string][]byte{
		"a.jpg": []byte("fake jpeg a"),
	})
	l, err := NewLoaderWithPassword(path, "unused")
	if err != nil {
		t.Fatalf("NewLoaderWithPassword failed: %v", err)
	}
	if l.TotalImage() != 1 {
		t.Errorf("Expected 1 image, got %d", l.TotalImage())
	}
	if err := l.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestArchiveLoader_TarGz(t *testing.T) {
	tmpDir := t.TempDir()
	path := writeTarGzArchive(t, tmpDir, map[string][]byte{
		"a.jpg": []byte("fake jpeg a"),
		"b.png": []byte("fake png b"),
	})
	assertArchiveNavigation(t, path)
}

func TestArchiveLoader_ParentFolderLink(t *testing.T) {
	tmpDir := t.TempDir()
	path := writeZipArchive(t, tmpDir, map[string][]byte{
		"a.jpg": []byte("fake jpeg a"),
		"b.png": []byte("fake png b"),
	})
	os.WriteFile(filepath.Join(tmpDir, "other.zip"), []byte("zip"), 0644)

	loader := NewArchiveLoader()
	if err := loader.Load(path); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	folders := loader.ChildFolders()
	if len(folders) != 1 || folders[0].Name != ".." {
		t.Fatalf("ChildFolders = %+v, want ..", folders)
	}
	if folders[0].FilePath != tmpDir {
		t.Fatalf(".. path = %s, want %s", folders[0].FilePath, tmpDir)
	}
	if got := loader.ChildArchives(); len(got) != 0 {
		t.Fatalf("ChildArchives = %+v, want none inside an archive", got)
	}
	if err := loader.OpenFolder(tmpDir); err == nil {
		t.Fatal("OpenFolder should fail so the viewer can switch to folder mode")
	}
}

func TestIsArchiveFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"photos.zip", true},
		{"album.tar.gz", true},
		{"comic.cbz", true},
		{"notes.txt", false},
		{"photo.jpg", false},
		{"photo.jpg.gz", false},
	}
	for _, tt := range tests {
		if got := isArchiveFile(tt.name); got != tt.want {
			t.Errorf("isArchiveFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
