package ui

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

// DropZone is a rounded panel inviting the user to drop a GIF, styled like
// goKomfy's drop zone: rounded input background, border that lights up on
// hover and flashes on drop.
type DropZone struct {
	widget.BaseWidget
	mu       sync.Mutex
	label    *widget.Label
	icon     *widget.Icon
	hovered  bool
	blinking bool
	timer    *time.Timer
}

// NewDropZone builds the empty-state drop zone.
func NewDropZone(text string) *DropZone {
	d := &DropZone{
		label: widget.NewLabel(text),
		icon:  widget.NewIcon(theme.MediaPhotoIcon()),
	}
	d.label.Alignment = fyne.TextAlignCenter
	d.label.Wrapping = fyne.TextWrapWord
	d.label.TextStyle = fyne.TextStyle{Italic: true}
	d.ExtendBaseWidget(d)
	return d
}

func (d *DropZone) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(theme.InputBackgroundColor())
	bg.CornerRadius = 12

	border := canvas.NewRectangle(color.Transparent)
	border.StrokeWidth = 2
	border.StrokeColor = theme.DisabledColor()

	return &dropZoneRenderer{
		dropZone: d,
		bg:       bg,
		border:   border,
		objects:  []fyne.CanvasObject{bg, border, d.icon, d.label},
	}
}

type dropZoneRenderer struct {
	dropZone *DropZone
	bg       *canvas.Rectangle
	border   *canvas.Rectangle
	objects  []fyne.CanvasObject
}

func (r *dropZoneRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.border.Resize(size)

	iconSize := float32(48)
	r.dropZone.icon.Resize(fyne.NewSize(iconSize, iconSize))
	r.dropZone.icon.Move(fyne.NewPos((size.Width-iconSize)/2, (size.Height-iconSize)/2-20))

	labelWidth := size.Width - 40
	r.dropZone.label.Resize(fyne.NewSize(labelWidth, 40))
	r.dropZone.label.Move(fyne.NewPos(20, (size.Height-iconSize)/2+iconSize-10))
}

func (r *dropZoneRenderer) MinSize() fyne.Size {
	return fyne.NewSize(380, 170)
}

func (d *DropZone) isBlinkingOrHovered() (bool, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.blinking, d.hovered
}

func (r *dropZoneRenderer) Refresh() {
	blinking, hovered := r.dropZone.isBlinkingOrHovered()
	switch {
	case blinking:
		r.bg.FillColor = theme.PrimaryColor()
		r.border.StrokeColor = theme.PrimaryColor()
	case hovered:
		r.border.StrokeColor = theme.PrimaryColor()
		r.border.StrokeWidth = 3
		r.bg.FillColor = theme.HoverColor()
	default:
		r.border.StrokeColor = theme.SeparatorColor()
		r.border.StrokeWidth = 2
		r.bg.FillColor = theme.InputBackgroundColor()
	}
	r.bg.Refresh()
	r.border.Refresh()
	r.dropZone.label.Refresh()
}

func (r *dropZoneRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *dropZoneRenderer) Destroy() {}

// MouseIn implements desktop.Hoverable.
func (d *DropZone) MouseIn(*desktop.MouseEvent) {
	d.mu.Lock()
	d.hovered = true
	d.mu.Unlock()
	d.Refresh()
}

// MouseMoved implements desktop.Hoverable.
func (d *DropZone) MouseMoved(*desktop.MouseEvent) {}

// MouseOut implements desktop.Hoverable.
func (d *DropZone) MouseOut() {
	d.mu.Lock()
	d.hovered = false
	d.mu.Unlock()
	d.Refresh()
}

// Flash briefly highlights the zone, used when a file is dropped on the window.
func (d *DropZone) Flash() {
	d.mu.Lock()
	d.blinking = true
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(150*time.Millisecond, func() {
		d.mu.Lock()
		d.blinking = false
		d.timer = nil
		d.mu.Unlock()
		fyne.Do(func() {
			d.Refresh()
		})
	})
	d.mu.Unlock()
	d.Refresh()
}
