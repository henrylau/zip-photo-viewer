package helper

import (
	"bytes"
	"fmt"
	"image"
	"path"
	"photoviewer/loader"
	"strings"

	"github.com/gen2brain/avif"
	"golang.org/x/image/webp"
)

var sizes = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}

func FormatFileSize(size int64) string {
	s := float64(size)
	unitsLimit := len(sizes)
	i := 0
	for s >= 1024 && i < unitsLimit {
		s = s / 1024
		i++
	}

	f := "%.0f %s"
	if i > 1 {
		f = "%.2f %s"
	}

	return fmt.Sprintf(f, s, sizes[i])
}

func LoadImage(data []byte, fileInfo loader.FileInfo) (image.Image, error) {
	reader := bytes.NewReader(data)
	switch strings.ToLower(path.Ext(fileInfo.Name)) {
	case ".jpg", ".jpeg", "png":
		img, _, err := image.Decode(reader)
		return img, err
	case ".webp":
		return webp.Decode(reader)
	case ".avif":
		return avif.Decode(reader)
	}
	return nil, fmt.Errorf("image format invalid")
}
