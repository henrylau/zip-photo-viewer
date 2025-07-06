package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	"log"
	"os"
	"photoviewer/helper"
	"photoviewer/loader"

	// "photoviewer/vips"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/x/explorer"
)

func main() {
	flag.Parse()
	file := flag.Arg(0)
	if file == "" {
		panic("Zip file not provided")
	}

	go func() {
		window := new(app.Window)
		err := run(window, file)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(window *app.Window, fileName string) error {
	var fileLoader loader.Loader
	var img image.Image
	var ops op.Ops
	var scale float32 = 0   // Track scale (default to fit-to-window)
	var offset f32.Point    // Track drag offset
	var dragging bool       // Track drag state
	var dragStart f32.Point // Track drag start position
	var rotation int        // Track rotation in degrees (0, 90, 180, 270)
	var tag = new(bool)
	expl := explorer.NewExplorer(window)

	fileLoader, err := loader.NewLoader(fileName)
	if err != nil {
		log.Fatal(err)
	}

	window.Option(
		app.Title("Photo Preview"),
	)

	// Load initial image
	if data, err := fileLoader.Get(); err == nil {
		img, err = helper.LoadImage(data, fileLoader.GetInfo())
		if err != nil {
			log.Printf("Failed to decode image: %v", err)
		} else {
			updateTitle(window, fileLoader.GetInfo(), img, fileLoader)
		}
	}

	for {
		e := window.Event()
		expl.ListenEvents(e)

		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			ops.Reset()
			gtx := app.NewContext(&ops, e)

			// Handle keyboard events
			for {
				event, ok := gtx.Event(
					key.Filter{Name: key.NameEscape},
					key.Filter{Name: key.NameLeftArrow},
					key.Filter{Name: key.NameRightArrow},
					key.Filter{Name: "`"},
					key.Filter{Name: "1"},
					key.Filter{Name: "2"}, // Hotkey for actual size
					key.Filter{Name: "O"},
					key.Filter{Name: "R"}, // Hotkey for rotate
				)
				if !ok {
					break
				}
				switch event := event.(type) {
				case key.Event:
					if event.State == key.Press {
						switch event.Name {
						case key.NameEscape:
							return nil
						case key.NameRightArrow:
							t := time.Now()
							if data, err := fileLoader.Next(); err == nil {
								img, err = helper.LoadImage(data, fileLoader.GetInfo())
								if err != nil {
									log.Printf("Failed to decode next image: %v", err)
								}
								// offset = f32.Pt(0, 0) // Reset offset on new image
								rotation = 0 // Reset rotation on new image
								// update title
								updateTitle(window, fileLoader.GetInfo(), img, fileLoader)
							}
							fmt.Println("Decode image", time.Now().Sub(t))
						case key.NameLeftArrow:
							t := time.Now()
							if data, err := fileLoader.Prev(); err == nil {
								img, err = helper.LoadImage(data, fileLoader.GetInfo())
								if err != nil {
									log.Printf("Failed to decode previous image: %v", err)
								}
								// offset = f32.Pt(0, 0) // Reset offset on new image
								rotation = 0 // Reset rotation on new image
								// update title
								updateTitle(window, fileLoader.GetInfo(), img, fileLoader)
							}
							fmt.Println("Decode image", time.Now().Sub(t))
						case "`":
							if scale == 0.5 {
								scale = 0
								offset = f32.Pt(0, 0) // Reset offset when zooming
							} else {
								scale = 0.5
							}
						case "1":
							if scale == 1 {
								scale = 0
								offset = f32.Pt(0, 0) // Reset offset when zooming
							} else {
								scale = 1
							}
						case "2":
							if scale == 2 {
								scale = 0
								offset = f32.Pt(0, 0) // Reset offset when zooming
							} else {
								scale = 2
							}
						case "O":
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

								if err := fileLoader.Close(); err != nil {
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
								img, err = helper.LoadImage(data, newLoader.GetInfo())
								if err != nil {
									fmt.Println(err)
									return
								}
								updateTitle(window, newLoader.GetInfo(), img, newLoader)
								fileLoader = newLoader
								window.Invalidate()
							}()
						case "R":
							rotation = (rotation + 90) % 360 // Rotate 90 degrees clockwise
							offset = f32.Pt(0, 0)            // Reset offset when zooming
						}
					}
				}
			}

			// Handle pointer events for dragging
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
							offset = offset.Add(delta)
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

			// Draw image
			if img != nil {
				drawImage(gtx, img, scale, offset, rotation)
			}

			e.Frame(gtx.Ops)
		}
	}
}

func updateTitle(window *app.Window, file loader.FileInfo, img image.Image, loader loader.Loader) {
	if img == nil {
		title := fmt.Sprintf("%s | %d/%d files | %s | PhotoViewer",
			file.Name, loader.Index()+1, loader.TotalImage(), helper.FormatFileSize(file.Size))
		window.Option(
			app.Title(title),
		)
	} else {
		title := fmt.Sprintf("%s | %d/%d files | %dx%d | %s | PhotoViewer",
			file.Name, loader.Index()+1, loader.TotalImage(), img.Bounds().Dx(), img.Bounds().Dy(), helper.FormatFileSize(file.Size))
		window.Option(
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

	// Apply transformations: scale, rotate, and translate
	// transform := f32.Affine2D{}.
	// 	Offset(f32.Pt(-imgWidth/2, -imgHeight/2)).
	// 	Scale(f32.Pt(0, 0), f32.Pt(scale, scale)).
	// 	Rotate(f32.Pt(0, 0), float32(rotation)*3.1415926535/180).
	// 	Offset(f32.Pt(float32(gtx.Constraints.Max.X)/2+offset.X, float32(gtx.Constraints.Max.Y)/2+offset.Y))

	transform := f32.Affine2D{}.
		Scale(f32.Pt(0, 0), f32.Pt(scale, scale)).
		Rotate(f32.Pt(imgWidth*scale/2, imgHeight*scale/2), float32(rotation)*3.1415926535/180).
		Offset(f32.Pt(-imgWidth*scale/2+float32(gtx.Constraints.Max.X)/2+offset.X, -imgHeight*scale/2+float32(gtx.Constraints.Max.Y)/2+offset.Y))

	op.Affine(transform).Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}
