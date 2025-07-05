package loader

import (
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/gen2brain/go-unarr"
)

type ZipLoader struct {
	zipFilePath string
	acceptedExt map[string]bool
	imageFiles  []*ImageFile
	current     *ImageFile
	currentIdx  int
	archive     *unarr.Archive
	cache       []CacheFile
	cacheLock   sync.Mutex
}

func (z *ZipLoader) Load(filePath string) error {
	archive, err := unarr.NewArchive(filePath)
	if err != nil {
		return err
	}

	z.archive = archive
	z.zipFilePath = filePath

	z.ScanFolder()
	return nil
}

func (z *ZipLoader) ScanFolder() error {
	if z.archive == nil {
		return nil
	}
	fmt.Println("Loading image")
	images := []*ImageFile{}

	for {
		err := z.archive.Entry()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
		ext := strings.ToLower(path.Ext(z.archive.Name()))
		if z.acceptedExt[ext] {
			image := &ImageFile{
				name:     path.Base(z.archive.Name()),
				filePath: z.archive.Name(),
				offset:   z.archive.Offset(),
				size:     int64(z.archive.Size()),
				modTime:  z.archive.ModTime(),
			}
			fmt.Println(image.name, image.size, image.modTime)
			images = append(images, image)
		} else {
			fmt.Println("Skip file", z.archive.Name())
		}
	}
	if len(images) == 0 {
		return fmt.Errorf("No Image found")
	}

	z.imageFiles = images

	z.sort()
	for _, i := range images {
		fmt.Println(i.filePath)
	}
	z.current = z.imageFiles[0]
	z.currentIdx = 0

	return nil
}

func (z *ZipLoader) sort() {
	// sort by name
	sort.Slice(z.imageFiles, func(a, b int) bool {
		return z.imageFiles[a].filePath < z.imageFiles[b].filePath
	})

	// calculate the new index
	if z.current != nil {
		for idx, i := range z.imageFiles {
			if i == z.current {
				z.currentIdx = idx
			}
		}
	}
}

func (z *ZipLoader) reset() {
	z.cacheLock.Lock()
	defer z.cacheLock.Unlock()

	z.zipFilePath = ""
	z.imageFiles = nil
	z.current = nil
	z.currentIdx = 0
	z.cache = make([]CacheFile, CACHE_SIZE)
}

func (z *ZipLoader) Close() error {
	z.reset()
	if z.archive != nil {
		err := z.archive.Close()
		return err
	}
	return nil
}

func (z *ZipLoader) Get() ([]byte, error) {
	if z.current == nil {
		return nil, fmt.Errorf("Image not loaded")
	}

	cacheIdx := z.currentIdx % len(z.cache)
	if z.cache[cacheIdx].index == z.currentIdx && len(z.cache[cacheIdx].data) != 0 {
		fmt.Println("Cache hit")
		defer func() {
			go z.preload()
		}()
		return z.cache[cacheIdx].data, nil
	}

	// load current file
	z.cacheLock.Lock()

	defer func() {
		z.cacheLock.Unlock()
		go z.preload()
	}()

	if err := z.archive.EntryAt(z.current.offset); err != nil {
		return nil, err
	}

	data, err := z.archive.ReadAll()

	if err != nil {
		return nil, err
	}

	z.cache[cacheIdx] = CacheFile{
		index: cacheIdx,
		data:  data,
	}
	return data, nil
}

func (z *ZipLoader) GetInfo() FileInfo {
	return FileInfo{
		Name:     z.current.name,
		FilePath: z.current.filePath,
		Size:     z.current.size,
		ModTime:  z.current.modTime,
	}
}

func (z *ZipLoader) preload() {
	if z.cacheLock.TryLock() {
		defer z.cacheLock.Unlock()
		fmt.Println("Preload start")
		idx := z.currentIdx
		cacheSize := len(z.cache)

		for i := 1; i < cacheSize/2 && idx+i < len(z.imageFiles); i++ {
			cacheIdx := (idx + i) % cacheSize
			if z.cache[cacheIdx].index != idx+i {
				z.archive.EntryAt(z.imageFiles[idx+i].offset)
				data, err := z.archive.ReadAll()
				if err != nil {
					// TODO: Log error
					fmt.Errorf("Extract file error", err)
					return
				}
				z.cache[cacheIdx] = CacheFile{
					index: idx + i,
					data:  data,
				}
				fmt.Println("Preload file : ", z.imageFiles[idx+i].name)
			}
		}

		fmt.Println("Preload complete")
	}
}

func (z *ZipLoader) Prev() ([]byte, error) {
	idx := (len(z.imageFiles) + z.currentIdx - 1) % len(z.imageFiles)
	z.current = z.imageFiles[idx]
	z.currentIdx = idx
	return z.Get()
}

func (z *ZipLoader) Next() ([]byte, error) {
	idx := (len(z.imageFiles) + z.currentIdx + 1) % len(z.imageFiles)
	z.current = z.imageFiles[idx]
	z.currentIdx = idx
	return z.Get()
}

func (z *ZipLoader) TotalImage() int {
	return len(z.imageFiles)
}

func (z *ZipLoader) Index() int {
	return z.currentIdx
}

func NewZipLoader() *ZipLoader {
	loader := ZipLoader{
		acceptedExt: map[string]bool{},
		cache:       make([]CacheFile, CACHE_SIZE),
	}

	for _, ext := range ACCEPTED_EXT {
		loader.acceptedExt[ext] = true
	}
	return &loader
}

var _ Loader = (*ZipLoader)(nil)
