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
								offset = f32.Pt(0, 0) // Reset offset on new image

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

								offset = f32.Pt(0, 0) // Reset offset on new image
								// update title
								updateTitle(window, fileLoader.GetInfo(), img, fileLoader)
							}
							fmt.Println("Decode image", time.Now().Sub(t))
						case "`":
							if scale == 0.5 {
								scale = 0
							} else {
								scale = 0.5
							}
							offset = f32.Pt(0, 0) // Reset offset when zooming
						case "1":
							if scale == 1 {
								scale = 0
							} else {
								scale = 1
							}
							offset = f32.Pt(0, 0) // Reset offset when zooming
						case "2":
							if scale == 2 {
								scale = 0
							} else {
								scale = 2
							}
							offset = f32.Pt(0, 0) // Reset offset when zooming
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
				// Calculate fit-to-window scale if scale is not set to actual size
				fitScaleX, fitScaleY := float32(gtx.Constraints.Max.X)/float32(img.Bounds().Dx()), float32(gtx.Constraints.Max.Y)/float32(img.Bounds().Dy())
				drawScale := scale
				if scale == 0 {
					drawScale = min(fitScaleX, fitScaleY)
				}

				drawImage(gtx, img, drawScale, offset)
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

func drawImage(gtx layout.Context, img image.Image, scale float32, offset f32.Point) {
	t := time.Now()
	imageOp := paint.NewImageOp(img)
	imageOp.Filter = paint.FilterNearest
	imageOp.Add(gtx.Ops)
	imgWidth, imgHeight := float32(img.Bounds().Dx()), float32(img.Bounds().Dy())
	// Center the image with drag offset
	centerX := (float32(gtx.Constraints.Max.X) - imgWidth*scale) / 2
	centerY := (float32(gtx.Constraints.Max.Y) - imgHeight*scale) / 2
	transform := f32.Affine2D{}.
		Scale(f32.Pt(0, 0), f32.Pt(scale, scale)).
		Offset(f32.Pt(centerX+offset.X, centerY+offset.Y))
	op.Affine(transform).Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	fmt.Println("draw image", time.Now().Sub(t))
}
