package viewer

import (
	"fmt"
	"log"
	"os"
	"time"

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
			log.Printf("Failed to decode next image: %v", err)
		}
		v.SetImage(img, info)
	}
	fmt.Println("Decode image", time.Now().Sub(t))
}

func PrevImage(v *Viewer, _ *explorer.Explorer) {
	t := time.Now()

	if data, err := v.Loader().Prev(); err == nil {
		info := v.Loader().GetInfo()
		img, err := helper.LoadImage(data, info)
		if err != nil {
			log.Printf("Failed to decode previous image: %v", err)
		}
		v.SetImage(img, info)
	}
	fmt.Println("Decode image", time.Now().Sub(t))
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
			fmt.Println(err)
			return
		}
		file, _ := reader.(*os.File)
		fileName := file.Name()
		fmt.Println(file, fileName, reader)
		reader.Close()

		if err := v.Loader().Close(); err != nil {
			fmt.Println(err)
			return
		}

		newLoader, err := loader.NewLoader(fileName)
		if err != nil {
			fmt.Println(err)
			return
		}

		data, err := newLoader.Get()
		if err != nil {
			fmt.Println(err)
			return
		}
		img, err := helper.LoadImage(data, newLoader.GetInfo())
		if err != nil {
			fmt.Println(err)
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

// 	if data, err := v.fileLoader.Get(); err == nil {
// 		imageData, _ := exif.Decode(bytes.NewReader(data))
// 		fmt.Println(imageData)
// 	}
