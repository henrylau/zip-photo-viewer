package viewer

import (
	"fmt"
	"image/color"
	"path/filepath"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type messagePrompt struct {
	title string
	body  string
	click widget.Clickable
}

func (v *Viewer) ShowNoMedia(path string) {
	name := filepath.Base(path)
	if name == "" || name == "." {
		name = "This archive"
	}
	v.message = &messagePrompt{
		title: "No media found",
		body:  fmt.Sprintf("%s does not contain any supported images.", name),
	}
	if v.window != nil {
		v.window.Invalidate()
	}
}

func (v *Viewer) dismissMessage() bool {
	if v.message == nil {
		return false
	}
	v.message = nil
	return true
}

func (v *Viewer) layoutMessagePrompt(gtx layout.Context, th *material.Theme) {
	msg := v.message
	if msg == nil {
		return
	}
	if msg.click.Clicked(gtx) {
		v.message = nil
		return
	}

	msg.click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		fill(gtx, color.NRGBA{A: 180})
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
						return material.H6(th, msg.title).Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Body2(th, msg.body).Layout(gtx)
					}),
				)
			})
		})
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}
