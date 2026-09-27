[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/henrylau/zip-photo-viewer)](https://goreportcard.com/report/github.com/henrylau/zip-photo-viewer)


# Compress files photo viewer

Simple photo viewer written in Go, which supports viewing images in a folder or compressed files like zip/7z/tar without extracting them. By using `gioui` for GUI and image rendering, and [`mholt/archives`](https://github.com/mholt/archives) for archive handling.

### Screens
![Screenshot](/images/screen.gif)

### Usage
```
./zip-photo-viewer [--password SECRET] <archive-file>|<image-file>
```

`--password` is optional. Encrypted 7z/RAR archives can also be unlocked with an in-app dialog. The flag value is visible in the process list; prefer the dialog when that matters.

### Features
- View images in a folder, supported format jpeg,png,webp,avif.
- View images in an archive without extracting. Supported formats: zip, rar, 7z, tar, compressed tar (`.tar.gz`, `.tgz`, `.tar.bz2`, `.tar.xz`, `.tar.zst`, `.tar.lz4`, `.tar.sz`), and comic archives (`.cbz`, `.cbr`, `.cbt`, `.cb7`).
- Open password-protected 7z/RAR (and `.cb7`/`.cbr`). Password-protected zip/cbz is not supported.
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
- Large compressed tarballs may be slower to browse than zip/7z/rar because tar is sequential.
- Large images may take a long time to load.
