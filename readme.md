[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/henrylau/zip-photo-viewer)](https://goreportcard.com/report/github.com/henrylau/zip-photo-viewer)


# Compress files photo viewer

Simple photo viewer written in Go, which supports viewing images in a folder or compressed files like zip/7z without extracting them. By using `gioui` for GUI and image rendering, and `unarr` for archive handling.

### Screens
![Screenshot](/images/screen.gif)

### Usage
```
./zip-photo-viewer <zip-file>|<image-file>
```

### Features
- View images in a folder, supported format jpeg,png,webp,avif.
- View images in a zip/7z file without extracting.
- Cache images for faster loading.
- Preload next image for smoother navigation.

### Control
- Use `left` and `right` arrow keys to navigate through images.
- `ESC` to exit the viewer.
- `R` to rotate the current image.
- `O` to load image from different folder or archive files.
- `` ` `` `1` `2` to zoom in and out from `50%` `100%` and `200%`.

### Build
```
go build -o output/photo-viewer cmd/photo-viewer.go
```

### Known issues
- goreleaser config only able to run in macOS, CI/CD pipeline not able to build darwin binary.
- Large zip/7z files may failure to open.
- Large images may take a long time to load.
