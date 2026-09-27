package loader

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gookit/slog"
	"github.com/mholt/archives"
)

type ArchiveLoader struct {
	archivePath string
	password    string
	acceptedExt map[string]bool
	imageFiles  []*ImageFile
	current     *ImageFile
	currentIdx  int
	fsys        fs.FS
	cache       []CacheFile
	cacheLock   sync.Mutex
}

func openArchiveFS(ctx context.Context, filePath, password string) (fs.FS, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	format, _, err := archives.Identify(ctx, filepath.Base(filePath), f)
	if err != nil {
		return nil, err
	}
	format = withPassword(format, password)
	extractor, ok := format.(archives.Extractor)
	if !ok {
		return nil, fmt.Errorf("unsupported archive format")
	}
	return &archives.ArchiveFS{
		Path:    filePath,
		Format:  extractor,
		Context: ctx,
	}, nil
}

func (a *ArchiveLoader) Load(filePath string) error {
	return a.LoadWithPassword(filePath, "")
}

func (a *ArchiveLoader) LoadWithPassword(filePath, password string) error {
	fsys, err := openArchiveFS(context.Background(), filePath, password)
	if err != nil {
		return classifyPasswordError(err, password != "")
	}

	a.fsys = fsys
	a.archivePath = filePath
	a.password = password

	return classifyPasswordError(a.ScanFolder(), password != "")
}

func (a *ArchiveLoader) ScanFolder() error {
	if a.fsys == nil {
		return nil
	}

	slog.Infof("Archive File : %s", a.archivePath)
	images := []*ImageFile{}

	err := fs.WalkDir(a.fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(path.Ext(name))
		if !a.acceptedExt[ext] {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		images = append(images, &ImageFile{
			name:     path.Base(name),
			filePath: name,
			size:     info.Size(),
			modTime:  info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return err
	}
	if len(images) == 0 {
		return fmt.Errorf("No Image found")
	}

	a.imageFiles = images

	slog.Infof("Total %d files", len(images))

	a.sort()
	a.current = a.imageFiles[0]
	a.currentIdx = 0

	return nil
}

func (a *ArchiveLoader) sort() {
	sort.Slice(a.imageFiles, func(i, j int) bool {
		return a.imageFiles[i].filePath < a.imageFiles[j].filePath
	})

	if a.current != nil {
		for idx, img := range a.imageFiles {
			if img == a.current {
				a.currentIdx = idx
			}
		}
	}
}

func (a *ArchiveLoader) reset() {
	a.cacheLock.Lock()
	defer a.cacheLock.Unlock()

	a.archivePath = ""
	a.password = ""
	a.imageFiles = nil
	a.current = nil
	a.currentIdx = 0
	a.cache = make([]CacheFile, CACHE_SIZE)
}

func (a *ArchiveLoader) Close() error {
	a.reset()
	if closer, ok := a.fsys.(io.Closer); ok {
		a.fsys = nil
		return closer.Close()
	}
	a.fsys = nil
	return nil
}

func (a *ArchiveLoader) readFile(filePath string) ([]byte, error) {
	f, err := a.fsys.Open(filePath)
	if err != nil {
		return nil, classifyPasswordError(err, a.password != "")
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, classifyPasswordError(err, a.password != "")
	}
	return data, nil
}

func (a *ArchiveLoader) Get() ([]byte, error) {
	if a.current == nil {
		return nil, fmt.Errorf("image not loaded")
	}

	cacheIdx := a.currentIdx % len(a.cache)
	if a.cache[cacheIdx].index == a.currentIdx && len(a.cache[cacheIdx].data) != 0 {
		slog.Debugf("Cache hit, idx: %d name: %s", a.currentIdx, a.current.name)
		defer func() {
			go a.preload()
		}()
		return a.cache[cacheIdx].data, nil
	}

	a.cacheLock.Lock()

	defer func() {
		a.cacheLock.Unlock()
		go a.preload()
	}()

	data, err := a.readFile(a.current.filePath)
	if err != nil {
		return nil, err
	}

	a.cache[cacheIdx] = CacheFile{
		index: a.currentIdx,
		data:  data,
	}
	return data, nil
}

func (a *ArchiveLoader) GetInfo() FileInfo {
	return FileInfo{
		Name:     a.current.name,
		FilePath: a.current.filePath,
		Size:     a.current.size,
		ModTime:  a.current.modTime,
	}
}

func (a *ArchiveLoader) preload() {
	if a.cacheLock.TryLock() {
		defer a.cacheLock.Unlock()
		slog.Debug("Preload start")
		idx := a.currentIdx
		cacheSize := len(a.cache)

		for i := 1; i < cacheSize/2 && idx+i < len(a.imageFiles); i++ {
			cacheIdx := (idx + i) % cacheSize
			if a.cache[cacheIdx].index != idx+i {
				data, err := a.readFile(a.imageFiles[idx+i].filePath)
				if err != nil {
					slog.Errorf("Extract file error", err)
					return
				}
				a.cache[cacheIdx] = CacheFile{
					index: idx + i,
					data:  data,
				}
				slog.Debugf("Preload file: %s", a.imageFiles[idx+i].name)
			}
		}

		slog.Debug("Preload complete")
	}
}

func (a *ArchiveLoader) Prev() ([]byte, error) {
	idx := (len(a.imageFiles) + a.currentIdx - 1) % len(a.imageFiles)
	a.current = a.imageFiles[idx]
	a.currentIdx = idx
	return a.Get()
}

func (a *ArchiveLoader) Next() ([]byte, error) {
	idx := (len(a.imageFiles) + a.currentIdx + 1) % len(a.imageFiles)
	a.current = a.imageFiles[idx]
	a.currentIdx = idx
	return a.Get()
}

func (a *ArchiveLoader) TotalImage() int {
	return len(a.imageFiles)
}

func (a *ArchiveLoader) Index() int {
	return a.currentIdx
}

func NewArchiveLoader() *ArchiveLoader {
	loader := ArchiveLoader{
		acceptedExt: map[string]bool{},
		cache:       make([]CacheFile, CACHE_SIZE),
	}

	for _, ext := range ACCEPTED_EXT {
		loader.acceptedExt[ext] = true
	}
	return &loader
}

var _ Loader = (*ArchiveLoader)(nil)
