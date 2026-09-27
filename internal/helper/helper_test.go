// filepath: internal/helper/helper_test.go
package helper

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/henrylau/zip-photo-viewer/internal/loader"
)

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1 kB"},
		{1536, "2 kB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tt := range tests {
		got := FormatFileSize(tt.size)
		if got != tt.expected {
			t.Errorf("FormatFileSize(%d) = %q, want %q", tt.size, got, tt.expected)
		}
	}
}

func createTestImageBytes(ext string) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	buf := new(bytes.Buffer)
	switch ext {
	case ".jpg", ".jpeg":
		jpeg.Encode(buf, img, nil)
	case ".png":
		png.Encode(buf, img)
	}
	return buf.Bytes()
}

func TestLoadImage_SupportedFormats(t *testing.T) {
	tests := []struct {
		ext      string
		makeData func() []byte
	}{
		{".jpg", func() []byte { return createTestImageBytes(".jpg") }},
		{".jpeg", func() []byte { return createTestImageBytes(".jpeg") }},
		{".png", func() []byte { return createTestImageBytes(".png") }},
	}

	for _, tt := range tests {
		data := tt.makeData()
		fi := loader.FileInfo{Name: "test" + tt.ext}
		img, err := LoadImage(data, fi)
		if err != nil {
			t.Errorf("LoadImage failed for %s: %v", tt.ext, err)
		}
		if img == nil {
			t.Errorf("LoadImage returned nil image for %s", tt.ext)
		}
	}
}

func TestLoadImage_UnsupportedFormat(t *testing.T) {
	data := []byte("not an image")
	fi := loader.FileInfo{Name: "test.txt"}
	img, err := LoadImage(data, fi)
	if err == nil {
		t.Error("Expected error for unsupported format, got nil")
	}
	if img != nil {
		t.Error("Expected nil image for unsupported format")
	}
}

func TestLoadImage_InvalidImageData(t *testing.T) {
	fi := loader.FileInfo{Name: "test.jpg"}
	data := []byte("invalid image data")
	img, err := LoadImage(data, fi)
	if err == nil {
		t.Error("Expected error for invalid image data, got nil")
	}
	if img != nil {
		t.Error("Expected nil image for invalid image data")
	}
}
