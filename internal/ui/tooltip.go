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
	tipOffsetX  = float32(10)
	tipOffsetY  = float32(16)
	tipTextSize = float32(12)
)

// hoverButton is a goKomfy-style icon button with a hover tooltip. It embeds
// widget.Button by value and calls ExtendBaseWidget on itself, so state
// changes (Enable/Disable) repaint correctly. Fyne has no built-in tooltip
// widget, so the tip is drawn as a plain (non-interactive) overlay near the
// cursor after a short hover delay.
type hoverButton struct {
	widget.Button

	win         fyne.Window
	tip         string
	timer       *time.Timer
	tipObj      fyne.CanvasObject
	lastTap     fyne.Position
	triggerTime time.Time
}

// newHoverButton creates an icon-only button with the given tooltip text.
func newHoverButton(icon fyne.Resource, tip string, f func(), win fyne.Window) *hoverButton {
	h := &hoverButton{win: win, tip: tip}
	h.Importance = widget.LowImportance
	h.Icon = icon
	h.OnTapped = f
	h.ExtendBaseWidget(h)
	return h
}

// Tapped hides any visible tooltip immediately so it can never intercept
// further events or linger over dialogs. The real GL driver dispatches
// MouseDown (not Tapped) to composed widgets, so both paths trigger the
// same action; a guard prevents double firing from drivers that send both.
func (h *hoverButton) Tapped(e *fyne.PointEvent) {
	h.trigger(e)
}

// MouseDown implements desktop.Mouseable.
func (h *hoverButton) MouseDown(e *desktop.MouseEvent) {
	if e.Button != desktop.MouseButtonPrimary {
		return
	}
	if dbgOn {
		fmt.Fprintf(os.Stderr, "DBG hoverButton.MouseDown tip=%q disabled=%v\n", h.tip, h.Disabled())
	}
	abs := e.AbsolutePosition
	h.trigger(&fyne.PointEvent{Position: e.Position, AbsolutePosition: abs})
}

// MouseUp implements desktop.Mouseable.
func (h *hoverButton) MouseUp(*desktop.MouseEvent) {}

func (h *hoverButton) trigger(e *fyne.PointEvent) {
	if dbgOn {
		fmt.Fprintf(os.Stderr, "DBG hoverButton.trigger tip=%q disabled=%v\n", h.tip, h.Disabled())
	}
	if h.Disabled() || h.triggeredAt(e) {
		return
	}
	h.lastTap = e.Position
	h.triggerTime = time.Now()
	h.cancelTimer()
	h.hideTip()
	if h.OnTapped != nil {
		h.OnTapped()
	}
}

// triggeredAt guards against a driver delivering both MouseDown and Tapped
// for the same click on the same object at nearly the same position/time.
func (h *hoverButton) triggeredAt(e *fyne.PointEvent) bool {
	return !h.triggerTime.IsZero() &&
		h.lastTap == e.Position &&
		time.Since(h.triggerTime) < 50*time.Millisecond
}

func (h *hoverButton) MouseIn(e *desktop.MouseEvent) {
	if dbgOn {
		fmt.Fprintf(os.Stderr, "DBG hoverButton.MouseIn tip=%q abs=%v\n", h.tip, e.AbsolutePosition)
	}
	h.cancelTimer()
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
		// Do not nil here: the timer callback checks via Stop()'s bool.
	}
}

func (h *hoverButton) showTip(pos fyne.Position) {
	if dbgOn {
		fmt.Fprintf(os.Stderr, "DBG showTip %q at %v\n", h.tip, pos)
	}
	if h.tipObj != nil {
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

	// Keep the tip inside the window.
	max := h.win.Canvas().Size()
	x, y := pos.X+tipOffsetX, pos.Y+tipOffsetY
	if x+size.Width > max.Width-4 {
		x = max.Width - size.Width - 4
	}
	if y+size.Height > max.Height-4 {
		y = pos.Y - size.Height - 8
	}
	if x < 4 {
		x = 4
	}
	if y < 4 {
		y = 4
	}
	tip.Move(fyne.NewPos(x, y))

	h.tipObj = tip
	h.win.Canvas().Overlays().Add(tip)
}

func (h *hoverButton) hideTip() {
	if h.tipObj != nil {
		h.win.Canvas().Overlays().Remove(h.tipObj)
		h.tipObj = nil
	}
}
