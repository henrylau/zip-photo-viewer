[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/henrylau/zip-photo-viewer)](https://goreportcard.com/report/github.com/henrylau/zip-photo-viewer)


# Compress files photo viewer

Simple photo viewer written in Go, which supports viewing images in a folder or compressed files like zip/7z/tar without extracting them. By using `gioui` for GUI and image rendering, and [`mholt/archives`](https://github.com/mholt/archives) for archive handling.

### Screens
![Screenshot](/images/screen.gif)

### Usage
```
./zip-photo-viewer [--password SECRET] [<folder>|<archive-file>|<image-file>]
```

If no path is given, the same file picker as `O` opens so you can choose an image or archive. `--password` is optional. Encrypted 7z/RAR archives can also be unlocked with an in-app dialog. The flag value is visible in the process list; prefer the dialog when that matters.

### Features
- View images in a folder, supported format jpeg,png,webp,avif.
- View images in an archive without extracting. Supported formats: zip, rar, 7z, tar, compressed tar (`.tar.gz`, `.tgz`, `.tar.bz2`, `.tar.xz`, `.tar.zst`, `.tar.lz4`, `.tar.sz`), and comic archives (`.cbz`, `.cbr`, `.cbt`, `.cb7`).
- Open password-protected 7z/RAR (and `.cb7`/`.cbr`). Password-protected zip/cbz is not supported.
- Cache images for faster loading.
- Preload next image for smoother navigation.

### Control
- Use `left` and `right` arrow keys to navigate through images.
- `Space` to toggle the album overlay; click a file to jump to it, a folder that contains images or archives to open that album, or an archive file to load it. Inside an archive the list shows only that archive’s contents plus `..` to return to the parent folder. Archive folder rows jump to the first photo in that folder.
- `ESC` closes the album tree if it is open, otherwise exits the viewer.
- `R` to rotate the current image.
- `O` to load image from different folder or archive files.
- `` ` `` `1` `2` to zoom in and out from `50%` `100%` and `200%`.

### Build
```
go build -o output/photo-viewer cmd/photo-viewer.go
```

### Release
Push a version tag to build macOS arm64 and Windows x86_64 artifacts and attach them to a GitHub Release:

```
git tag v0.1.0
git push origin v0.1.0
```

The `Release` workflow also supports a manual run from the Actions tab (artifacts only, no GitHub Release). CI runs tests on macOS and Windows for pushes and pull requests to `main`.

### Known issues
- Large compressed tarballs may be slower to browse than zip/7z/rar because tar is sequential.
- Large images may take a long time to load.
