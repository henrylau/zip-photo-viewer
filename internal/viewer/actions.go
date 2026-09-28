package viewer

import (
	"errors"
	"os"
	"time"

	"github.com/gookit/slog"
	"github.com/henrylau/zip-photo-viewer/internal/helper"
	"github.com/henrylau/zip-photo-viewer/internal/loader"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/x/explorer"
)

func NextImage(v *Viewer, _ *explorer.Explorer) {
	if v.Loader() == nil {
		return
	}
	t := time.Now()

	if data, err := v.Loader().Next(); err == nil {
		info := v.Loader().GetInfo()
		img, err := helper.LoadImage(data, info)
		if err != nil {
			slog.Errorf("Failed to decode next image: %s", info.Name, err)
		}
		v.SetImage(img, info)
	}
	slog.Debugf("Decode image %s", time.Since(t))
}

func PrevImage(v *Viewer, _ *explorer.Explorer) {
	if v.Loader() == nil {
		return
	}
	t := time.Now()

	if data, err := v.Loader().Prev(); err == nil {
		info := v.Loader().GetInfo()
		img, err := helper.LoadImage(data, info)
		if err != nil {
			slog.Errorf("Failed to decode next image: %s", info.Name, err)
		}
		v.SetImage(img, info)
	}
	slog.Debugf("Decode image %s", time.Since(t))
}

func ScaleImage(scale float32) func(*Viewer, *explorer.Explorer) {
	return func(v *Viewer, _ *explorer.Explorer) {
		if v.Scale() == scale {
			v.SetScale(0)
		} else {
			v.SetScale(scale)
		}
	}
}

func LoadFile(v *Viewer, expl *explorer.Explorer) {
	go func() {
		exts := []string{}
		exts = append(exts, loader.ACCEPTED_EXT...)
		exts = append(exts, loader.PickerArchiveExt...)
		reader, err := expl.ChooseFile(exts...)

		if err != nil {
			slog.Error(err)
			return
		}
		file, _ := reader.(*os.File)
		fileName := file.Name()
		reader.Close()
		openSource(v, fileName)
	}()

}

func OpenAlbumArchive(v *Viewer, path string) {
	go openSource(v, path)
}

func openSource(v *Viewer, fileName string) {
	slog.Info("Load file: %s", fileName)
	newLoader, err := loader.NewLoader(fileName)
	if err != nil {
		handleOpenError(v, fileName, err)
		return
	}

	if err := v.applyLoader(newLoader); err != nil {
		_ = newLoader.Close()
		handleOpenError(v, fileName, err)
		return
	}
	v.sourcePath = fileName
}

func handleOpenError(v *Viewer, fileName string, err error) {
	switch {
	case loader.IsPasswordError(err):
		v.AskPassword(fileName, err)
	case errors.Is(err, loader.ErrNoMedia):
		v.ShowNoMedia(fileName)
	default:
		slog.Error(err)
	}
}

func RotateImage(v *Viewer, expl *explorer.Explorer) {
	v.SetRotation((v.Rotation() + 90) % 360)
	v.SetOffset(f32.Pt(0, 0))
}

type Action struct {
	Key     key.Name
	Handler func(*Viewer, *explorer.Explorer)
}

func LoadActions() []Action {
	return []Action{
		{
			Key:     key.NameRightArrow,
			Handler: NextImage,
		},
		{
			Key:     key.NameLeftArrow,
			Handler: PrevImage,
		},
		{
			Key:     "`",
			Handler: ScaleImage(0.5),
		},
		{
			Key:     "1",
			Handler: ScaleImage(1),
		},
		{
			Key:     "2",
			Handler: ScaleImage(2),
		},
		{
			Key:     "R",
			Handler: RotateImage,
		},
		{
			Key:     "O",
			Handler: LoadFile,
		},
		{
			Key:     key.NameSpace,
			Handler: ToggleAlbum,
		},
	}
}
