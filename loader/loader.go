package loader

import (
	"fmt"
	"path"
	"strings"
	"time"
)

var ACCEPTED_EXT = []string{".png", ".jpg", ".jpeg", ".avif", ".webp"}
var ZIP_EXT = []string{".zip", ".tar", ".rar", ".7z"}

const CACHE_SIZE = 10

type ImageFile struct {
	name     string
	filePath string
	// stat     fs.FileInfo
	offset  int64
	size    int64
	modTime time.Time
	isDir   bool
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

type Loader interface {
	Load(filePath string) error
	GetInfo() FileInfo
	Get() ([]byte, error)
	Prev() ([]byte, error)
	Next() ([]byte, error)
	TotalImage() int
	Index() int
	Close() error
}

func NewLoader(file string) (Loader, error) {
	switch {
	case isZipFile(file):
		loader := NewZipLoader()

		if err := loader.Load(file); err != nil {
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

func isZipFile(file string) bool {
	ext := strings.ToLower(path.Ext(file))
	for _, e := range ZIP_EXT {
		if e == ext {
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
