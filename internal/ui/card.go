package ui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var dbgCard = os.Getenv("QGIF_DEBUG") != ""

const (
	cardImageSize = float32(160) // logical px the thumbnail is shown at
	thumbMaxSide  = 320          // thumbnail pixels, 2× for HiDPI sharpness
	cardRadius    = float32(8)
)

// FrameCard is one thumbnail in the grid: rounded card background, image,
// caption and a selection border with a check icon.
//
// Interaction: single click displays the frame (preview + Save would save it),
// double click toggles selection, Shift+click selects a range.
type FrameCard struct {
	widget.BaseWidget

	index    int
	img      *canvas.Image
	label    *widget.Label
	border   *canvas.Rectangle
	bg       *canvas.Rectangle
	check    *widget.Icon
	onTap    func(index int, shift bool)
	mu       sync.Mutex
	hovered  bool
	selected bool
	current  bool
}

// newFrameCard creates a card. onTap receives the frame index and whether
// Shift was held; double-click detection happens in the UI layer.
func newFrameCard(onTap func(index int, shift bool)) *FrameCard {
	c := &FrameCard{onTap: onTap}

	c.bg = canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	c.bg.CornerRadius = cardRadius

	c.img = &canvas.Image{}
	c.img.FillMode = canvas.ImageFillContain
	c.img.SetMinSize(fyne.NewSize(cardImageSize, cardImageSize))

	c.label = widget.NewLabel("")
	c.label.Alignment = fyne.TextAlignCenter
	c.label.TextStyle = fyne.TextStyle{Monospace: true}

	c.border = canvas.NewRectangle(color.Transparent)
	c.border.CornerRadius = cardRadius

	c.check = widget.NewIcon(theme.ConfirmIcon())
	c.check.Resize(fyne.NewSize(22, 22))
	c.check.Move(fyne.NewPos(6, 6))
	c.check.Hidden = true

	c.ExtendBaseWidget(c)
	return c
}

// Set updates the card contents and visual state.
func (c *FrameCard) Set(index int, thumb *image.RGBA, selected, current bool, delayMs int) {
	c.index = index
	c.img.Image = thumb
	c.label.Text = fmt.Sprintf("#%04d · %d ms", index+1, delayMs)
	c.selected = selected
	c.current = current
	c.Refresh()
}

// CreateRenderer builds the layered visual: rounded background, selection
// border, image above caption, and the check icon in the top-left corner.
func (c *FrameCard) CreateRenderer() fyne.WidgetRenderer {
	imageAndCaption := container.NewBorder(nil, c.label, nil, nil, c.img)
	stack := container.NewStack(c.bg, c.border, imageAndCaption, container.NewWithoutLayout(c.check))
	return &frameCardRenderer{card: c, stack: stack}
}

type frameCardRenderer struct {
	card  *FrameCard
	stack *fyne.Container
}

func (r *frameCardRenderer) Layout(size fyne.Size)        { r.stack.Resize(size) }
func (r *frameCardRenderer) MinSize() fyne.Size           { return r.stack.MinSize() }
func (r *frameCardRenderer) Objects() []fyne.CanvasObject { return r.stack.Objects }
func (r *frameCardRenderer) Destroy()                     {}

func (r *frameCardRenderer) Refresh() {
	r.card.applyState()
	r.stack.Refresh()
}

// MouseDown triggers the tap callback; double-click detection happens in the
// UI layer using timing, so every press reports here.
func (c *FrameCard) MouseDown(e *desktop.MouseEvent) {
	if dbgCard {
		fmt.Fprintf(os.Stderr, "DBG card.MouseDown idx=%d\n", c.index)
	}
	if e.Button == desktop.MouseButtonPrimary && c.onTap != nil {
		c.onTap(c.index, e.Modifier&fyne.KeyModifierShift != 0)
	}
}

// MouseUp is required by desktop.Mouseable.
func (c *FrameCard) MouseUp(*desktop.MouseEvent) {}

// MouseIn implements desktop.Hoverable for the hover highlight.
func (c *FrameCard) MouseIn(*desktop.MouseEvent) {
	c.mu.Lock()
	c.hovered = true
	c.mu.Unlock()
	c.Refresh()
}

// MouseMoved implements desktop.Hoverable.
func (c *FrameCard) MouseMoved(*desktop.MouseEvent) {}

// MouseOut implements desktop.Hoverable.
func (c *FrameCard) MouseOut() {
	c.mu.Lock()
	c.hovered = false
	c.mu.Unlock()
	c.Refresh()
}

// applyState picks the border colours: selected wins with primary + check,
// the viewed frame gets the focus outline, hover gets a subtle highlight.
func (c *FrameCard) applyState() {
	c.mu.Lock()
	hovered := c.hovered
	c.mu.Unlock()

	c.check.Hidden = !c.selected
	switch {
	case c.selected:
		c.border.StrokeColor = theme.Color(theme.ColorNamePrimary)
		c.border.StrokeWidth = 3
	case c.current:
		c.border.StrokeColor = theme.Color(theme.ColorNameFocus)
		c.border.StrokeWidth = 2
	case hovered:
		c.border.StrokeColor = theme.Color(theme.ColorNameHover)
		c.border.StrokeWidth = 2
	default:
		c.border.StrokeWidth = 0
	}
	c.bg.Refresh()
	c.border.Refresh()
	c.check.Refresh()
	c.img.Refresh()
	c.label.Refresh()
}
