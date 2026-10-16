package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mamorett/qgif/internal/gifdec"
)

// Preview shows the last clicked frame at full resolution with its metadata,
// on a goKomfy-style rounded card.
type Preview struct {
	Container fyne.CanvasObject

	card  *canvas.Rectangle
	img   *canvas.Image
	label *widget.Label
}

// NewPreview builds the preview panel. It starts empty.
func NewPreview() *Preview {
	card := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	card.CornerRadius = 8

	img := &canvas.Image{}
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScalePixels

	label := widget.NewLabel("")
	label.Alignment = fyne.TextAlignCenter
	label.TextStyle = fyne.TextStyle{Monospace: true}
	label.Truncation = fyne.TextTruncateEllipsis

	inner := container.NewBorder(nil, label, nil, nil, img)
	c := container.NewStack(card, container.NewPadded(inner))

	return &Preview{Container: c, card: card, img: img, label: label}
}

// ShowFrame displays a full-resolution frame. total is the frame count.
func (p *Preview) ShowFrame(f gifdec.Frame, total int) {
	p.img.Image = f.Image
	p.img.Refresh()
	p.label.Text = fmt.Sprintf("Frame %d / %d\nDelay %d ms\n%d×%d",
		f.Index+1, total, f.DelayMs, f.Image.Bounds().Dx(), f.Image.Bounds().Dy())
	p.label.Refresh()
}

// Clear hides any shown frame.
func (p *Preview) Clear() {
	p.img.Image = nil
	p.img.Refresh()
	p.label.Text = ""
	p.label.Refresh()
}
