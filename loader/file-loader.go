package loader

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
	cache       []CacheFile
	cacheLock   sync.Mutex
}

func NewFileLoader() *FileLoader {
	loader := &FileLoader{
		acceptedExt: map[string]bool{},
		cache:       make([]CacheFile, CACHE_SIZE),
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
	f.cache = make([]CacheFile, CACHE_SIZE)
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

	// scan
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

	fmt.Println(f.basePath, f.currentFile)
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
				fmt.Println("path", filePath)
			} else {
				// TODO: Log error message
			}
		} else if e.IsDir() {
			childDir = append(childDir, e.Name())
		}
	}

	f.imageFiles = images
	f.childDir = childDir

	f.sort()
	// load the file
	if f.basePath == f.currentFile && len(f.imageFiles) > 0 {
		f.current = f.imageFiles[0]
		f.currentIdx = 0
	}

	fmt.Println(f.current)
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

		cacheIdx := f.currentIdx % len(f.cache)
		if f.cache[cacheIdx].index == f.currentIdx && len(f.cache[cacheIdx].data) != 0 {
			fmt.Println("Cache hit")
			defer func() {
				go f.preload()
			}()
			return f.cache[cacheIdx].data, nil
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

		f.cache[cacheIdx] = CacheFile{
			index: cacheIdx,
			data:  data,
		}

		return data, nil
	}
	return nil, fmt.Errorf("Image not loaded")
}

func (f *FileLoader) preload() {
	if f.cacheLock.TryLock() {
		defer f.cacheLock.Unlock()
		fmt.Println("Preload start")
		idx := f.currentIdx
		cacheSize := len(f.cache)

		for i := 1; i < cacheSize/2 && idx+i < len(f.imageFiles); i++ {
			cacheIdx := (idx + i) % cacheSize
			if f.cache[cacheIdx].index != idx+i {
				data, err := os.ReadFile(f.imageFiles[idx+i].filePath)
				if err != nil {
					// TODO: Log error
					fmt.Errorf("Extract file error", err)
					return
				}
				f.cache[cacheIdx] = CacheFile{
					index: idx + i,
					data:  data,
				}
				fmt.Println("Preload file : ", f.imageFiles[idx+i].name)
			}
		}

		fmt.Println("Preload complete")
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
