package viewer

import (
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
		exts = append(exts, loader.ZIP_EXT...)
		reader, err := expl.ChooseFile(exts...)

		if err != nil {
			slog.Error(err)
			return
		}
		file, _ := reader.(*os.File)
		fileName := file.Name()
		slog.Info("Load file: %s", file.Name)
		reader.Close()

		if err := v.Loader().Close(); err != nil {
			slog.Error(err)
			return
		}

		newLoader, err := loader.NewLoader(fileName)
		if err != nil {
			slog.Error(err)
			return
		}

		data, err := newLoader.Get()
		if err != nil {
			slog.Error(err)
			return
		}
		img, err := helper.LoadImage(data, newLoader.GetInfo())
		if err != nil {
			slog.Error(err)
			return
		}

		v.SetLoader(newLoader)
		v.SetImage(img, newLoader.GetInfo())
	}()

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
			Handler: NextImage,
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
	}
}
