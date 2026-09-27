package viewer

import (
	"errors"
	"image"
	"image/color"

	"github.com/henrylau/zip-photo-viewer/internal/helper"
	"github.com/henrylau/zip-photo-viewer/internal/loader"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type passwordPrompt struct {
	path      string
	editor    widget.Editor
	unlock    widget.Clickable
	cancel    widget.Clickable
	errText   string
	unlocking bool
	focused   bool
}

func (v *Viewer) AskPassword(path string, err error) {
	errText := ""
	if err != nil && errors.Is(err, loader.ErrBadPassword) {
		errText = "incorrect password"
	}
	v.prompt = &passwordPrompt{
		path: path,
		editor: widget.Editor{
			SingleLine: true,
			Submit:     true,
			Mask:       '*',
		},
		errText: errText,
	}
	if v.window != nil {
		v.window.Invalidate()
	}
}

func (v *Viewer) cancelPassword() bool {
	v.prompt = nil
	return v.fileLoader == nil
}

func (v *Viewer) submitPassword() {
	if v.prompt == nil || v.prompt.unlocking {
		return
	}
	password := v.prompt.editor.Text()
	path := v.prompt.path
	v.prompt.unlocking = true
	v.prompt.errText = ""
	go v.openWithPassword(path, password)
}

func (v *Viewer) setPromptError(err error) {
	if v.prompt == nil {
		return
	}
	v.prompt.unlocking = false
	if loader.IsPasswordError(err) {
		v.prompt.errText = "incorrect password"
	} else {
		v.prompt.errText = err.Error()
	}
	if v.window != nil {
		v.window.Invalidate()
	}
}

func (v *Viewer) applyLoader(newLoader loader.Loader) error {
	data, err := newLoader.Get()
	if err != nil {
		return err
	}
	img, err := helper.LoadImage(data, newLoader.GetInfo())
	if err != nil {
		return err
	}
	old := v.fileLoader
	v.SetLoader(newLoader)
	v.SetImage(img, newLoader.GetInfo())
	if old != nil {
		_ = old.Close()
	}
	if v.window != nil {
		v.window.Invalidate()
	}
	return nil
}

func (v *Viewer) openWithPassword(path, password string) {
	newLoader, err := loader.NewLoaderWithPassword(path, password)
	if err != nil {
		v.setPromptError(err)
		return
	}
	if err := v.applyLoader(newLoader); err != nil {
		_ = newLoader.Close()
		v.setPromptError(err)
		return
	}
	v.sourcePath = path
	v.prompt = nil
	if v.window != nil {
		v.window.Invalidate()
	}
}

func (v *Viewer) layoutPasswordPrompt(gtx layout.Context, th *material.Theme) (shouldExit bool) {
	if v.prompt == nil {
		return false
	}

	submitted := false
	for {
		ev, ok := v.prompt.editor.Update(gtx)
		if !ok {
			break
		}
		if _, isSubmit := ev.(widget.SubmitEvent); isSubmit {
			submitted = true
		}
	}
	if v.prompt.cancel.Clicked(gtx) {
		return v.cancelPassword()
	}
	if !v.prompt.unlocking && (v.prompt.unlock.Clicked(gtx) || submitted) {
		v.submitPassword()
	}
	if v.prompt == nil {
		return false
	}

	fill(gtx, color.NRGBA{A: 180})

	if !v.prompt.focused {
		gtx.Execute(key.FocusCmd{Tag: &v.prompt.editor})
		v.prompt.focused = true
	}

	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		maxW := gtx.Dp(400)
		if pad := gtx.Constraints.Max.X - gtx.Dp(40); pad > 0 && pad < maxW {
			maxW = pad
		}
		gtx.Constraints.Min.X = maxW
		gtx.Constraints.Max.X = maxW

		return card(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.H6(th, "Archive password").Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Body2(th, "This archive is encrypted. Enter the password to view images.").Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					ed := material.Editor(th, &v.prompt.editor, "Password")
					return ed.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if v.prompt.errText == "" {
						return layout.Dimensions{}
					}
					lbl := material.Body2(th, v.prompt.errText)
					lbl.Color = color.NRGBA{R: 180, A: 255}
					return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, lbl.Layout)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceStart}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Button(th, &v.prompt.cancel, "Cancel").Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := "Unlock"
							if v.prompt.unlocking {
								label = "Unlocking..."
							}
							return material.Button(th, &v.prompt.unlock, label).Layout(gtx)
						}),
					)
				}),
			)
		})
	})
	return false
}

func fill(gtx layout.Context, c color.NRGBA) {
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

func card(gtx layout.Context, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(20)).Layout(gtx, w)
	call := macro.Stop()

	defer clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(8)).Push(gtx.Ops).Pop()
	paint.ColorOp{Color: color.NRGBA{R: 250, G: 250, B: 250, A: 255}}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	call.Add(gtx.Ops)
	return dims
}
