package viewer

import (
	"fmt"
	"image"
	"log"
	"os"

	"github.com/henrylau/zip-photo-viewer/internal/helper"
	"github.com/henrylau/zip-photo-viewer/internal/loader"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
)

type Viewer struct {
	window     *app.Window
	fileLoader loader.Loader
	sourcePath string
	image      image.Image
	rotation   int
	scale      float32
	offset     f32.Point
	prompt     *passwordPrompt
}

func NewViewer(fileLoader loader.Loader, sourcePath string) Viewer {
	return Viewer{
		fileLoader: fileLoader,
		sourcePath: sourcePath,
	}
}

func (v *Viewer) Main() {
	v.window = new(app.Window)
	go func() {
		err := v.run()
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func (v *Viewer) SetImage(image image.Image, info loader.FileInfo) {
	v.image = image
	v.updateTitle(info, image)
}

func (v *Viewer) SetRotation(rotation int) {
	v.rotation = rotation
}
func (v *Viewer) Rotation() int {
	return v.rotation
}

func (v *Viewer) Scale() float32 {
	return v.scale
}

func (v *Viewer) SetScale(s float32) {
	v.scale = s
}

func (v *Viewer) Loader() loader.Loader {
	return v.fileLoader
}
func (v *Viewer) SetLoader(loader loader.Loader) {
	v.fileLoader = loader
}

func (v *Viewer) Offset() f32.Point {
	return v.Offset()
}

func (v *Viewer) SetOffset(o f32.Point) {
	v.offset = o
}

func (v *Viewer) run() error {
	var ops op.Ops
	// var scale float32 = 0   // Track scale (default to fit-to-window)
	// var offset f32.Point    // Track drag offset
	var dragging bool       // Track drag state
	var dragStart f32.Point // Track drag start position
	// var rotation int        // Track rotation in degrees (0, 90, 180, 270)
	var tag = new(bool)
	expl := explorer.NewExplorer(v.window)

	v.window.Option(
		app.Title("Photo Preview"),
	)

	th := material.NewTheme()

	// Load initial image
	if v.fileLoader != nil {
		data, err := v.fileLoader.Get()
		if loader.IsPasswordError(err) {
			v.AskPassword(v.sourcePath, err)
		} else if err == nil {
			v.image, err = helper.LoadImage(data, v.fileLoader.GetInfo())
			if err != nil {
				log.Printf("Failed to decode image: %v", err)
			} else {
				v.updateTitle(v.fileLoader.GetInfo(), v.image)
			}
		} else {
			log.Printf("Failed to load image: %v", err)
		}
	}

	actions := LoadActions()
	eventFilters := []event.Filter{key.Filter{Name: key.NameEscape}}
	for _, action := range actions {
		eventFilters = append(eventFilters, key.Filter{
			Name: action.Key,
		})
	}

	for {
		e := v.window.Event()
		expl.ListenEvents(e)

		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			ops.Reset()
			gtx := app.NewContext(&ops, e)

			// Handle keyboard events
			for {
				event, ok := gtx.Event(eventFilters...)
				if !ok {
					break
				}
				switch event := event.(type) {
				case key.Event:
					if event.State == key.Press {
						if event.Name == key.NameEscape {
							if v.prompt != nil {
								if v.cancelPassword() {
									return nil
								}
								break
							}
							return nil
						}

						if v.prompt != nil {
							break
						}

						for _, action := range actions {
							if action.Key == event.Name {
								action.Handler(v, expl)
							}
						}
					}
				}
			}

			// Handle pointer events for dragging
			if v.prompt == nil {
				for {
					event, ok := gtx.Event(
						pointer.Filter{
							Target: tag,
							Kinds:  pointer.Press | pointer.Drag | pointer.Release,
						},
					)
					if !ok {
						break
					}
					switch event := event.(type) {
					case pointer.Event:
						switch event.Kind {
						case pointer.Press:
							dragging = true
							dragStart = event.Position
						case pointer.Drag:
							if dragging {
								delta := event.Position.Sub(dragStart)
								v.offset = v.offset.Add(delta)
								dragStart = event.Position
							}
						case pointer.Release:
							dragging = false
						}
					}
				}

				// Register to listen for pointer Drag events.
				pr := clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops)
				event.Op(gtx.Ops, tag)
				pr.Pop()
			}

			// Draw image
			if v.image != nil {
				drawImage(gtx, v.image, v.scale, v.offset, v.rotation)
			}

			if v.layoutPasswordPrompt(gtx, th) {
				return nil
			}

			e.Frame(gtx.Ops)
		}
	}
}

func (v *Viewer) updateTitle(file loader.FileInfo, img image.Image) {
	if v.fileLoader == nil {
		return
	}
	if img == nil {
		title := fmt.Sprintf("%s | %d/%d files | %s | PhotoViewer",
			file.Name, v.fileLoader.Index()+1, v.fileLoader.TotalImage(), helper.FormatFileSize(file.Size))
		v.window.Option(
			app.Title(title),
		)
	} else {
		title := fmt.Sprintf("%s | %d/%d files | %dx%d | %s | PhotoViewer",
			file.Name, v.fileLoader.Index()+1, v.fileLoader.TotalImage(), img.Bounds().Dx(), img.Bounds().Dy(), helper.FormatFileSize(file.Size))
		v.window.Option(
			app.Title(title),
		)
	}
}

func drawImage(gtx layout.Context, img image.Image, scale float32, offset f32.Point, rotation int) {
	imageOp := paint.NewImageOp(img)
	imageOp.Filter = paint.FilterNearest
	imageOp.Add(gtx.Ops)
	imgWidth, imgHeight := float32(img.Bounds().Dx()), float32(img.Bounds().Dy())

	// Adjust dimensions for rotation
	var drawWidth, drawHeight float32
	switch rotation {
	case 90, 270:
		drawWidth, drawHeight = imgHeight, imgWidth
	default:
		drawWidth, drawHeight = imgWidth, imgHeight
	}

	if scale == 0 {
		// Calculate fit-to-window scale if scale is not set to actual size
		fitScaleX, fitScaleY := float32(gtx.Constraints.Max.X)/drawWidth, float32(gtx.Constraints.Max.Y)/drawHeight
		scale = min(fitScaleX, fitScaleY)
	}

	transform := f32.Affine2D{}.
		Scale(f32.Pt(0, 0), f32.Pt(scale, scale)).
		Rotate(f32.Pt(imgWidth*scale/2, imgHeight*scale/2), float32(rotation)*3.1415926535/180).
		Offset(f32.Pt(-imgWidth*scale/2+float32(gtx.Constraints.Max.X)/2+offset.X, -imgHeight*scale/2+float32(gtx.Constraints.Max.Y)/2+offset.Y))

	op.Affine(transform).Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}
