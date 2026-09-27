package loader

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gookit/slog"
)

type FileLoader struct {
	basePath    string
	currentFile string
	isFolder    bool

	acceptedExt map[string]bool
	imageFiles  []*ImageFile
	childDir    []string
	current     *ImageFile
	currentIdx  int
	cacher      Cacher
	cacheLock   sync.Mutex
}

func NewFileLoader() *FileLoader {
	loader := &FileLoader{
		acceptedExt: map[string]bool{},
		cacher:      NewCacher(CACHE_SIZE),
	}

	for _, ext := range ACCEPTED_EXT {
		loader.acceptedExt[ext] = true
	}
	return loader
}

func (f *FileLoader) reset() {
	f.cacheLock.Lock()
	defer f.cacheLock.Unlock()

	f.basePath = ""
	f.currentFile = ""
	f.imageFiles = nil
	f.childDir = nil
	f.current = nil
	f.currentIdx = 0
	f.cacher.Clear()
}

func (f *FileLoader) Load(fileName string) error {
	fileName, _ = filepath.Abs(fileName)
	info, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	f.isFolder = info.IsDir()
	if f.isFolder {
		f.basePath = fileName
		f.currentFile = fileName

	} else {
		f.basePath = filepath.Dir(fileName)
		f.currentFile = fileName
	}

	f.cacher.Clear()
	f.current = nil
	f.currentIdx = 0
	f.ScanFolder()

	return nil
}

func (f *FileLoader) ScanFolder() {
	entries, err := os.ReadDir(f.basePath)
	if err != nil {
		f.imageFiles = []*ImageFile{}
		return
	}

	images := []*ImageFile{}
	childDir := []string{}

	slog.Infof("Path : %s", f.basePath)
	for _, e := range entries {
		if !e.IsDir() && f.acceptedExt[strings.ToLower(path.Ext(e.Name()))] {
			filePath := filepath.Join(f.basePath, e.Name())
			stat, err := os.Stat(filePath)
			if err == nil {
				imageFile := &ImageFile{
					name:     e.Name(),
					filePath: filePath,
					size:     stat.Size(),
					modTime:  stat.ModTime(),
					isDir:    stat.IsDir(),
				}
				images = append(images, imageFile)
				if f.currentFile == filePath {
					f.current = imageFile
				}
			} else {
				slog.Errorf("Unable log file info: %s", filePath, err)
			}
		} else if e.IsDir() {
			childDir = append(childDir, e.Name())
		}
	}
	slog.Infof("Total %d files", len(images))

	f.imageFiles = images
	f.childDir = childDir

	f.sort()
	// load the file
	if f.basePath == f.currentFile && len(f.imageFiles) > 0 {
		f.current = f.imageFiles[0]
		f.currentIdx = 0
	}
}

func (f *FileLoader) sort() {
	// sort by name
	sort.Slice(f.imageFiles, func(a, b int) bool {
		return f.imageFiles[a].name < f.imageFiles[b].name
	})
	for idx, i := range f.imageFiles {
		if i == f.current {
			f.currentIdx = idx
		}
	}
}

func (f *FileLoader) Get() ([]byte, error) {
	if f.current != nil {

		slot := f.currentIdx % f.cacher.Size
		if c, err := f.cacher.Get(slot); err == nil && c.index == f.currentIdx && c.data != nil {
			slog.Debugf("Cache hit, idx: %d name: %s", c.index, f.current.name)
			defer func() {
				go f.preload()
			}()
			return c.data, nil
		}

		// load current file
		f.cacheLock.Lock()

		defer func() {
			f.cacheLock.Unlock()
			go f.preload()
		}()

		data, err := os.ReadFile(f.current.filePath)
		if err != nil {
			return nil, err
		}

		f.cacher.Set(slot, CacheFile{
			index: slot,
			data:  data,
		})

		return data, nil
	}
	return nil, fmt.Errorf("image not loaded")
}

func (f *FileLoader) preload() {
	if f.cacheLock.TryLock() {
		defer f.cacheLock.Unlock()
		slog.Debug("Preload start")
		idx := f.currentIdx

		for i := 1; i < f.cacher.Size/2 && idx+i < len(f.imageFiles); i++ {
			slot := (idx + i) % f.cacher.Size
			if c, err := f.cacher.Get(slot); err == nil && c.index != idx+1 {
				data, err := os.ReadFile(f.imageFiles[idx+i].filePath)
				if err != nil {
					// TODO: Log error
					slog.Errorf("Extract file error", err)
					return
				}
				f.cacher.Set(slot, CacheFile{
					index: idx + i,
					data:  data,
				})
				slog.Debugf("Preload file: %s", f.imageFiles[idx+i].name)
			}
		}

		slog.Debug("Preload complete")
	}
}

func (f *FileLoader) Next() ([]byte, error) {
	idx := (f.currentIdx + 1) % len(f.imageFiles)
	f.current = f.imageFiles[idx]
	f.currentIdx = idx
	f.currentFile = f.current.filePath
	return f.Get()
}

func (f *FileLoader) Prev() ([]byte, error) {
	idx := (len(f.imageFiles) + f.currentIdx - 1) % len(f.imageFiles)
	f.current = f.imageFiles[idx]
	f.currentIdx = idx
	f.currentFile = f.current.filePath
	return f.Get()
}

func (f *FileLoader) Seek(index int) ([]byte, error) {
	if index < 0 || index >= len(f.imageFiles) {
		return nil, fmt.Errorf("index out of range")
	}
	f.current = f.imageFiles[index]
	f.currentIdx = index
	f.currentFile = f.current.filePath
	return f.Get()
}

func dirHasAlbumContent(dir string, accepted map[string]bool) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if accepted[strings.ToLower(path.Ext(name))] || isArchiveFile(name) {
			return true
		}
	}
	return false
}

func dirHasAlbumFolders(dir string, accepted map[string]bool) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() && dirHasAlbumContent(filepath.Join(dir, e.Name()), accepted) {
			return true
		}
	}
	return false
}

func listArchivesInDir(dir, skip string) []FileInfo {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	archives := []FileInfo{}
	skip = filepath.Clean(skip)
	for _, e := range entries {
		if e.IsDir() || !isArchiveFile(e.Name()) {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if skip != "" && filepath.Clean(p) == skip {
			continue
		}
		archives = append(archives, FileInfo{Name: e.Name(), FilePath: p})
	}
	sort.Slice(archives, func(i, j int) bool {
		return archives[i].Name < archives[j].Name
	})
	return archives
}

func (f *FileLoader) ChildFolders() []FileInfo {
	folders := []FileInfo{}
	parent := filepath.Dir(f.basePath)
	if parent != f.basePath && (dirHasAlbumContent(parent, f.acceptedExt) || dirHasAlbumFolders(parent, f.acceptedExt)) {
		folders = append(folders, FileInfo{Name: "..", FilePath: parent})
	}

	names := append([]string{}, f.childDir...)
	sort.Strings(names)
	for _, name := range names {
		p := filepath.Join(f.basePath, name)
		if dirHasAlbumContent(p, f.acceptedExt) {
			folders = append(folders, FileInfo{Name: name, FilePath: p})
		}
	}
	return folders
}

func (f *FileLoader) ChildArchives() []FileInfo {
	return listArchivesInDir(f.basePath, "")
}

func (f *FileLoader) OpenFolder(path string) error {
	return f.Load(path)
}

func (f *FileLoader) Entries() []FileInfo {
	entries := make([]FileInfo, len(f.imageFiles))
	for i, img := range f.imageFiles {
		entries[i] = FileInfo{
			Name:     img.name,
			FilePath: img.filePath,
			Size:     img.size,
			ModTime:  img.modTime,
		}
	}
	return entries
}

func (f *FileLoader) TotalImage() int {
	return len(f.imageFiles)
}

func (f *FileLoader) Index() int {
	return f.currentIdx
}

func (f *FileLoader) Close() error {
	f.reset()
	return nil
}

func (f *FileLoader) GetInfo() FileInfo {
	return FileInfo{
		Name:     f.current.name,
		FilePath: f.current.filePath,
		Size:     f.current.size,
		ModTime:  f.current.modTime,
	}
}

var _ Loader = (*FileLoader)(nil)
