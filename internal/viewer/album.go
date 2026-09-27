package viewer

import (
	"fmt"
	"image"
	"image/color"

	"github.com/gookit/slog"
	"github.com/henrylau/zip-photo-viewer/internal/helper"
	"github.com/henrylau/zip-photo-viewer/internal/loader"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
)

const albumPanelWidth = 320

var (
	albumDim       = color.NRGBA{R: 20, G: 20, B: 22, A: 140}
	albumPanelBg   = color.NRGBA{R: 48, G: 48, B: 50, A: 230}
	albumHeaderFg  = color.NRGBA{R: 245, G: 245, B: 247, A: 255}
	albumMutedFg   = color.NRGBA{R: 170, G: 170, B: 176, A: 255}
	albumRowFg     = color.NRGBA{R: 228, G: 228, B: 232, A: 255}
	albumFolderFg  = color.NRGBA{R: 196, G: 196, B: 202, A: 255}
	albumCurrentBg = color.NRGBA{R: 255, G: 200, B: 90, A: 48}
	albumCurrentFg = color.NRGBA{R: 255, G: 226, B: 150, A: 255}
	albumDivider   = color.NRGBA{R: 255, G: 255, B: 255, A: 22}
)

func ToggleAlbum(v *Viewer, _ *explorer.Explorer) {
	if v.Loader() == nil {
		return
	}
	v.albumOpen = !v.albumOpen
	if v.albumOpen {
		v.rebuildAlbumTree()
	}
}

func OpenAlbumFolder(v *Viewer, dir string) {
	if v.Loader() == nil {
		return
	}
	if err := v.Loader().OpenFolder(dir); err != nil {
		newLoader, nerr := loader.NewLoader(dir)
		if nerr != nil {
			slog.Errorf("Open folder failed: %v", nerr)
			return
		}
		old := v.fileLoader
		v.SetLoader(newLoader)
		if old != nil {
			_ = old.Close()
		}
	}
	v.sourcePath = dir
	if data, err := v.Loader().Get(); err != nil {
		slog.Errorf("Load folder image failed: %v", err)
	} else {
		info := v.Loader().GetInfo()
		img, err := helper.LoadImage(data, info)
		if err != nil {
			slog.Errorf("Failed to decode image: %s", info.Name, err)
		} else {
			v.SetImage(img, info)
		}
	}
	v.rebuildAlbumTree()
	if v.window != nil {
		v.window.Invalidate()
	}
}

func JumpImage(v *Viewer, index int) {
	if v.Loader() == nil {
		return
	}
	data, err := v.Loader().Seek(index)
	if err != nil {
		slog.Errorf("Seek image failed: %v", err)
		return
	}
	info := v.Loader().GetInfo()
	img, err := helper.LoadImage(data, info)
	if err != nil {
		slog.Errorf("Failed to decode image: %s", info.Name, err)
		return
	}
	v.SetImage(img, info)
}

func (v *Viewer) rebuildAlbumTree() {
	v.albumTree = nil
	v.albumExpanded = map[string]bool{}
	v.albumHighlight = -1
	v.albumSource = nil
	if v.fileLoader == nil {
		return
	}
	v.albumTree = buildAlbumTree(v.fileLoader.Entries(), v.fileLoader.ChildFolders(), v.fileLoader.ChildArchives())
	v.albumSource = v.fileLoader
	v.expandPathToIndex(v.fileLoader.Index())
	v.albumHighlight = v.fileLoader.Index()
}

func (v *Viewer) ensureAlbumTree() {
	if v.fileLoader == nil {
		v.albumTree = nil
		v.albumSource = nil
		return
	}
	if v.albumSource != v.fileLoader {
		v.rebuildAlbumTree()
		return
	}
	idx := v.fileLoader.Index()
	if v.albumHighlight != idx {
		v.expandPathToIndex(idx)
		v.albumHighlight = idx
	}
}

func (v *Viewer) expandPathToIndex(index int) {
	if v.albumExpanded == nil {
		v.albumExpanded = map[string]bool{}
	}
	for _, p := range findNodePath(v.albumTree, index) {
		v.albumExpanded[p] = true
	}
}

func (v *Viewer) layoutAlbumOverlay(gtx layout.Context, th *material.Theme) layout.Dimensions {
	fill(gtx, albumDim)

	width := gtx.Dp(albumPanelWidth)
	if max := gtx.Constraints.Max.X * 2 / 5; max > 0 && width > max {
		width = max
	}
	maxH := gtx.Constraints.Max.Y - gtx.Dp(32)
	if capH := gtx.Dp(560); maxH > capH {
		maxH = capH
	}
	if maxH < 0 {
		maxH = gtx.Constraints.Max.Y
	}

	return layout.Inset{
		Top:   unit.Dp(16),
		Right: unit.Dp(16),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.NE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = width
			gtx.Constraints.Max.X = width
			gtx.Constraints.Min.Y = 0
			gtx.Constraints.Max.Y = maxH
			return albumPanel(gtx, func(gtx layout.Context) layout.Dimensions {
				return v.layoutAlbumContent(gtx, th)
			})
		})
	})
}

func (v *Viewer) layoutAlbumContent(gtx layout.Context, th *material.Theme) layout.Dimensions {
	v.ensureAlbumTree()
	rows := flattenAlbumTree(v.albumTree, v.albumExpanded)
	if len(v.albumClicks) != len(rows) {
		v.albumClicks = make([]widget.Clickable, len(rows))
	}

	total := 0
	current := 0
	if v.fileLoader != nil {
		total = v.fileLoader.TotalImage()
		current = v.fileLoader.Index() + 1
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.H6(th, "Album")
						lbl.Color = albumHeaderFg
						return lbl.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Caption(th, fmt.Sprintf("%d / %d", current, total))
						lbl.Color = albumMutedFg
						return lbl.Layout(gtx)
					}),
				)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(1))}.Push(gtx.Ops).Pop()
				paint.ColorOp{Color: albumDivider}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(1))}
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			list := material.List(th, &v.albumList)
			return list.Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
				return v.layoutAlbumRow(gtx, th, rows[i], &v.albumClicks[i])
			})
		}),
	)
}

func (v *Viewer) layoutAlbumRow(gtx layout.Context, th *material.Theme, row treeRow, clk *widget.Clickable) layout.Dimensions {
	if clk.Clicked(gtx) {
		switch {
		case row.node.archivePath != "":
			OpenAlbumArchive(v, row.node.archivePath)
		case row.node.dirPath != "":
			OpenAlbumFolder(v, row.node.dirPath)
		case row.node.index >= 0:
			JumpImage(v, row.node.index)
		default:
			v.albumExpanded[row.node.path] = true
			if i := firstLeafIndex(row.node); i >= 0 {
				JumpImage(v, i)
			}
		}
	}

	label := row.node.name
	fg := albumRowFg
	if row.node.archivePath != "" {
		fg = albumFolderFg
	} else if row.node.dirPath != "" {
		fg = albumFolderFg
		label = "▸  " + label
	} else if row.node.index < 0 {
		fg = albumFolderFg
		if v.albumExpanded[row.node.path] {
			label = "▾  " + label
		} else {
			label = "▸  " + label
		}
	}

	current := v.fileLoader != nil && row.node.index == v.fileLoader.Index()
	if current {
		fg = albumCurrentFg
	}

	return clk.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{
			Left:   unit.Dp(10 + 16*row.depth),
			Right:  unit.Dp(10),
			Top:    unit.Dp(8),
			Bottom: unit.Dp(8),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body1(th, label)
			lbl.Color = fg
			return lbl.Layout(gtx)
		})
		if dims.Size.X < gtx.Constraints.Max.X {
			dims.Size.X = gtx.Constraints.Max.X
		}
		call := macro.Stop()

		if current {
			defer clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(8)).Push(gtx.Ops).Pop()
			paint.ColorOp{Color: albumCurrentBg}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
		}
		call.Add(gtx.Ops)
		return dims
	})
}

func albumPanel(gtx layout.Context, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(16)).Layout(gtx, w)
	call := macro.Stop()

	r := gtx.Dp(14)
	defer clip.UniformRRect(image.Rectangle{Max: dims.Size}, r).Push(gtx.Ops).Pop()
	paint.ColorOp{Color: albumPanelBg}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	call.Add(gtx.Ops)
	return dims
}
