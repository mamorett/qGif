package ui

import (
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	tipDelay    = 400 * time.Millisecond
	tipPad      = float32(6)
	tipTextSize = float32(12)
)

// hoverButton is an icon button with a hover tooltip. It embeds widget.Button
// by value and calls ExtendBaseWidget on itself, so state changes (Enable/Disable)
// repaint correctly.
//
// Tooltips are rendered into a dedicated overlay layer in the window's content
// container rather than Canvas().Overlays() so that they remain strictly visual
// and never block mouse clicks across the canvas.
type hoverButton struct {
	widget.Button

	win       fyne.Window
	tipLayer  *fyne.Container
	tip       string
	timer     *time.Timer
	tipObj    fyne.CanvasObject
	lastInRel fyne.Position // cursor inside the widget at MouseIn
	lastInAbs fyne.Position // cursor on the canvas at MouseIn
}

// newHoverButton creates an icon-only button with the given tooltip text.
func newHoverButton(icon fyne.Resource, tip string, f func(), win fyne.Window, tipLayer *fyne.Container) *hoverButton {
	h := &hoverButton{win: win, tipLayer: tipLayer, tip: tip}
	h.Importance = widget.LowImportance
	h.Icon = icon
	h.OnTapped = f
	h.ExtendBaseWidget(h)
	return h
}

// newHoverButtonWithText creates a button with text and icon and a hover tooltip.
func newHoverButtonWithText(text string, icon fyne.Resource, tip string, f func(), win fyne.Window, tipLayer *fyne.Container) *hoverButton {
	h := &hoverButton{win: win, tipLayer: tipLayer, tip: tip}
	h.Importance = widget.LowImportance
	h.Text = text
	h.Icon = icon
	h.OnTapped = f
	h.ExtendBaseWidget(h)
	return h
}

// Tapped hides any visible tooltip immediately and dispatches the standard button tap.
func (h *hoverButton) Tapped(e *fyne.PointEvent) {
	h.cancelTimer()
	h.hideTip()
	h.Button.Tapped(e)
}

func (h *hoverButton) MouseIn(e *desktop.MouseEvent) {
	if dbgOn {
		fmt.Fprintf(os.Stderr, "DBG hoverButton.MouseIn tip=%q abs=%v\n", h.tip, e.AbsolutePosition)
	}
	h.cancelTimer()
	h.lastInRel = e.Position
	h.lastInAbs = e.AbsolutePosition
	pos := e.AbsolutePosition
	h.timer = time.AfterFunc(tipDelay, func() {
		fyne.Do(func() {
			h.timer = nil
			h.showTip(pos)
		})
	})
}

func (h *hoverButton) MouseMoved(*desktop.MouseEvent) {}

func (h *hoverButton) MouseOut() {
	h.cancelTimer()
	h.hideTip()
}

func (h *hoverButton) cancelTimer() {
	if h.timer != nil {
		h.timer.Stop()
	}
}

func (h *hoverButton) showTip(pos fyne.Position) {
	if dbgOn {
		fmt.Fprintf(os.Stderr, "DBG showTip %q at %v\n", h.tip, pos)
	}
	if h.tipObj != nil || h.tipLayer == nil {
		return
	}
	text := canvas.NewText(h.tip, theme.Color(theme.ColorNameForeground))
	text.TextSize = tipTextSize
	text.TextStyle = fyne.TextStyle{Monospace: true}

	tw, th := text.MinSize().Width, text.MinSize().Height
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.StrokeColor = theme.Color(theme.ColorNameDisabled)
	bg.StrokeWidth = 1
	bg.CornerRadius = 6

	size := fyne.NewSize(tw+2*tipPad, th+2*tipPad)
	bg.Resize(size)
	text.Move(fyne.NewPos(tipPad, tipPad))
	text.Resize(fyne.NewSize(tw, th))
	text.Refresh()

	tip := container.NewWithoutLayout(bg, text)
	tip.Resize(size)

	// Anchor below the button (or above for footer buttons).
	max := h.win.Canvas().Size()
	btnLeft := pos.X - h.lastInRel.X
	btnTop := pos.Y - h.lastInRel.Y
	btnBottom := btnTop + h.Size().Height
	x, y := btnLeft, btnBottom+4
	if y+size.Height > max.Height-4 {
		y = btnTop - size.Height - 4 // footer buttons: show above instead
	}
	if x+size.Width > max.Width-4 {
		x = max.Width - size.Width - 4
	}
	if x < 4 {
		x = 4
	}
	if y < 4 {
		y = 4
	}
	tip.Move(fyne.NewPos(x, y))

	h.tipObj = tip
	h.tipLayer.Objects = []fyne.CanvasObject{tip}
	h.tipLayer.Refresh()
}

func (h *hoverButton) hideTip() {
	if h.tipObj != nil {
		if h.tipLayer != nil {
			h.tipLayer.Objects = nil
			h.tipLayer.Refresh()
		}
		h.tipObj = nil
	}
}
