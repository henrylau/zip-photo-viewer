package loader

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

var ErrNoMedia = errors.New("no media files found")

var ACCEPTED_EXT = []string{".png", ".jpg", ".jpeg", ".avif", ".webp"}

// ARCHIVE_EXT is ordered longest-suffix first so .tar.gz is not treated as .gz.
var ARCHIVE_EXT = []string{
	".tar.bz2",
	".tar.lz4",
	".tar.zst",
	".tar.gz",
	".tar.xz",
	".tar.sz",
	".tbz2",
	".tzst",
	".txz",
	".tgz",
	".cb7",
	".cbr",
	".cbt",
	".cbz",
	".tar",
	".rar",
	".7z",
	".zip",
}

// PickerArchiveExt is ARCHIVE_EXT plus last-component filters so OS dialogs
// can show compressed-tar files (.tar.gz appears as .gz).
var PickerArchiveExt = append([]string{".gz", ".bz2", ".xz", ".zst", ".lz4", ".sz"}, ARCHIVE_EXT...)

const CACHE_SIZE = 10

type ImageFile struct {
	name     string
	filePath string
	size     int64
	modTime  time.Time
	isDir    bool
}

type FileInfo struct {
	Name     string
	FilePath string
	Size     int64
	ModTime  time.Time
}
type CacheFile struct {
	data  []byte
	index int
}

type Cacher struct {
	cacheItems []CacheFile
	Size       int
}

func NewCacher(size int) Cacher {
	return Cacher{
		cacheItems: make([]CacheFile, size),
		Size:       size,
	}
}

func (cacher *Cacher) Get(slot int) (CacheFile, error) {
	if slot >= cacher.Size {
		return CacheFile{}, fmt.Errorf("Cacher index out of bound")
	}
	return cacher.cacheItems[slot], nil
}
func (cacher *Cacher) Set(slot int, cache CacheFile) {
	cacher.cacheItems[slot] = cache
}
func (cacher *Cacher) Clear() {
	clear(cacher.cacheItems)
}

type Loader interface {
	Load(filePath string) error
	GetInfo() FileInfo
	Get() ([]byte, error)
	Prev() ([]byte, error)
	Next() ([]byte, error)
	Seek(index int) ([]byte, error)
	Entries() []FileInfo
	ChildFolders() []FileInfo
	ChildArchives() []FileInfo
	OpenFolder(path string) error
	TotalImage() int
	Index() int
	Close() error
}

func NewLoader(file string) (Loader, error) {
	return NewLoaderWithPassword(file, "")
}

func NewLoaderWithPassword(file, password string) (Loader, error) {
	switch {
	case isDir(file):
		loader := NewFileLoader()

		if err := loader.Load(file); err != nil {
			return nil, err
		}
		return loader, nil
	case isArchiveFile(file):
		loader := NewArchiveLoader()

		if err := loader.LoadWithPassword(file, password); err != nil {
			return nil, err
		}
		return loader, nil
	case isImageFile(file):
		loader := NewFileLoader()

		if err := loader.Load(file); err != nil {
			return nil, err
		}
		return loader, nil
	}
	return nil, fmt.Errorf("FileFormatNotAllowed")
}

func isDir(file string) bool {
	info, err := os.Stat(file)
	return err == nil && info.IsDir()
}

func isArchiveFile(file string) bool {
	lower := strings.ToLower(file)
	for _, suf := range ARCHIVE_EXT {
		if strings.HasSuffix(lower, suf) {
			return true
		}
	}
	return false
}

func isImageFile(file string) bool {
	ext := strings.ToLower(path.Ext(file))
	for _, e := range ACCEPTED_EXT {
		if e == ext {
			return true
		}
	}
	return false
}
