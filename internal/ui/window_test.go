package ui

import (
	"image"
	"image/color"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/mamorett/qgif/internal/gifdec"
	"github.com/mamorett/qgif/internal/selection"
)

func testFrames(n int) []gifdec.Frame {
	out := make([]gifdec.Frame, n)
	for i := range out {
		img := image.NewRGBA(image.Rect(0, 0, 4, 4))
		img.Set(0, 0, color.RGBA{byte(i), 0, 0, 255})
		out[i] = gifdec.Frame{Index: i, Image: img, DelayMs: 100}
	}
	return out
}

func newLoadedApp(t *testing.T, n int) *App {
	t.Helper()
	a := test.NewApp()
	w := test.NewWindow(nil)
	t.Cleanup(func() { w.Close(); a.Quit() })

	u := New(a, w)
	u.frames = testFrames(n)
	u.thumbImgs = make([]*image.RGBA, n)
	u.delays = make([]int, n)
	for i := range u.frames {
		u.thumbImgs[i] = u.frames[i].Image
		u.delays[i] = 100
	}
	u.sel = selection.New(n)
	u.info = gifdec.Info{Width: 4, Height: 4, Frames: n}
	return u
}

func TestThemeApplied(t *testing.T) {
	a := test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	New(a, w)
	if _, ok := a.Settings().Theme().(*qgifTheme); !ok {
		t.Fatalf("theme not applied, got %T", a.Settings().Theme())
	}
}

func TestSingleClickViewsDoubleClickSelects(t *testing.T) {
	u := newLoadedApp(t, 5)

	var taps []int
	card := newFrameCard(func(i int, shift bool) {
		taps = append(taps, i)
		u.onCardTap(i, shift)
	})
	card.Set(0, u.thumbImgs[0], false, false, 100)

	click := func() { card.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary}) }

	click() // single click → view only
	if u.current != 0 || u.sel.Count() != 0 || len(taps) != 1 {
		t.Fatalf("single click: current=%d count=%d taps=%v", u.current, u.sel.Count(), taps)
	}
	if u.exportBtn.Disabled() {
		t.Error("export must be enabled when a frame is viewed")
	}

	click() // second click within the window → select
	if u.sel.Count() != 1 || !u.sel.IsSelected(0) {
		t.Fatalf("double click did not select: count=%d", u.sel.Count())
	}

	click() // third click resets to single (view); selection stays
	if u.sel.Count() != 1 {
		t.Fatalf("selection lost on next single click: count=%d", u.sel.Count())
	}
	if u.preview == nil {
		t.Fatal("preview nil")
	}
}

func TestShiftClickSelectsRange(t *testing.T) {
	u := newLoadedApp(t, 5)
	var taps []int
	card := newFrameCard(func(i int, shift bool) {
		taps = append(taps, i)
		u.onCardTap(i, shift)
	})
	card.Set(0, u.thumbImgs[0], false, false, 100)

	// Select frame 0 via double click, then Shift+click frame 3 → 0..3.
	card.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary})
	card.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary})
	if u.sel.Count() != 1 {
		t.Fatalf("precondition: count=%d", u.sel.Count())
	}
	card.index = 3
	card.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary, Modifier: fyne.KeyModifierShift})
	if u.sel.Count() != 4 {
		t.Fatalf("range selection count = %d, want 4", u.sel.Count())
	}
}

func TestExportIndicesFallBackToViewed(t *testing.T) {
	u := newLoadedApp(t, 3)
	if got := u.exportIndices(); got != nil {
		t.Fatalf("exportIndices with nothing = %v, want nil", got)
	}
	u.current = 2
	if got := u.exportIndices(); len(got) != 1 || got[0] != 2 {
		t.Fatalf("exportIndices with no selection = %v, want [2]", got)
	}
	u.sel.Toggle(0)
	u.sel.Toggle(2)
	got := u.exportIndices()
	if len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("exportIndices with selection = %v, want [0 2]", got)
	}
}

func TestHoverTooltipShowsAndHides(t *testing.T) {
	u := newLoadedApp(t, 1)
	btn := u.openBtn

	btn.showTip(fyne.NewPos(10, 10))
	if len(u.tooltipLayer.Objects) == 0 {
		t.Fatal("tooltip not shown in tooltipLayer")
	}
	if u.w.Canvas().Overlays().Top() != nil {
		t.Fatal("tooltip should not be placed in Canvas().Overlays() to avoid blocking clicks")
	}
	btn.hideTip()
	if len(u.tooltipLayer.Objects) != 0 {
		t.Fatal("tooltip not removed from tooltipLayer")
	}

	// MouseOut must clear a pending timer so a late show cannot appear.
	btn.MouseIn(&desktop.MouseEvent{})
	btn.MouseOut()
	time.Sleep(600 * time.Millisecond)
	if len(u.tooltipLayer.Objects) != 0 {
		t.Fatal("tooltip appeared after MouseOut")
	}
}


func TestCardHoverChangesBorder(t *testing.T) {
	u := newLoadedApp(t, 2)
	card := newFrameCard(func(int, bool) {})
	card.Set(0, u.thumbImgs[0], false, false, 100)

	card.MouseIn(&desktop.MouseEvent{})
	if card.border.StrokeColor == nil || card.border.StrokeWidth != 2 {
		t.Fatalf("hover border not applied: w=%v", card.border.StrokeWidth)
	}
	card.MouseOut()
	if card.border.StrokeWidth != 0 {
		t.Fatalf("hover border not cleared: w=%v", card.border.StrokeWidth)
	}
}

func TestCardSelectedAndCurrentStates(t *testing.T) {
	u := newLoadedApp(t, 2)
	card := newFrameCard(func(int, bool) {})

	card.Set(0, u.thumbImgs[0], true, true, 100)
	if card.border.StrokeWidth != 3 || card.check.Hidden {
		t.Fatalf("selected look wrong: w=%v checkHidden=%v", card.border.StrokeWidth, card.check.Hidden)
	}
	card.Set(0, u.thumbImgs[0], false, true, 100)
	if card.border.StrokeWidth != 2 || !card.check.Hidden {
		t.Fatalf("current look wrong: w=%v checkHidden=%v", card.border.StrokeWidth, card.check.Hidden)
	}
	card.Set(0, u.thumbImgs[0], false, false, 100)
	if card.border.StrokeWidth != 0 {
		t.Fatalf("plain card should have no border: w=%v", card.border.StrokeWidth)
	}
}

// TestShiftClickAfterSingleClickIsTheRegressionGuard: the natural multi-pick
// flow (click frame 0, Shift+click frame 2) must select the whole range even
// though no double-click/toggle set the anchor first.
func TestShiftClickAfterSingleClickIsTheRegressionGuard(t *testing.T) {
	u := newLoadedApp(t, 5)
	u.onCardTap(0, false)
	u.onCardTap(2, true)
	if got := u.sel.Indices(); len(got) != 3 || got[0] != 0 || got[2] != 2 {
		t.Fatalf("expected range 0..2 selected, got %v", got)
	}
	if got := u.exportIndices(); len(got) != 3 {
		t.Fatalf("Save must export all 3 frames, got %v", got)
	}
}

func TestUnloadResetsState(t *testing.T) {
	u := newLoadedApp(t, 3)
	u.lastGifPath = "/tmp/x/sample.gif"
	u.onCardTap(1, false)
	u.sel.Toggle(1)
	u.unload()

	if len(u.frames) != 0 || u.sel != nil || u.current != -1 || u.lastGifPath != "" {
		t.Fatalf("state not cleared: frames=%d current=%d path=%q", len(u.frames), u.current, u.lastGifPath)
	}
	if u.grid.Visible() || !u.empty.Visible() {
		t.Fatal("expected drop zone back, grid hidden")
	}
	if u.status.Text != "No GIF loaded" {
		t.Fatalf("status = %q", u.status.Text)
	}
	if !u.exportBtn.Disabled() {
		t.Fatal("export must be disabled after unload")
	}
	u.unload() // idempotent no-op
}

func TestExportButtonHitTest(t *testing.T) {
	u := newLoadedApp(t, 5)
	u.w.Resize(fyne.NewSize(800, 600))
	u.onCardTap(0, false) // enables exportBtn
	t.Logf("exportBtn pos=%v size=%v disabled=%v", u.exportBtn.Position(), u.exportBtn.Size(), u.exportBtn.Disabled())

	u.exportBtn.showTip(fyne.NewPos(770, 20))

	c := u.w.Canvas()
	btnPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(u.exportBtn)
	btnSize := u.exportBtn.Size()
	t.Logf("exportBtn absolute pos=%v size=%v", btnPos, btnSize)

	misses := 0
	hits := 0
	// Sample across the entire button area
	for x := btnPos.X + 2; x < btnPos.X+btnSize.Width-2; x += 4 {
		for y := btnPos.Y + 2; y < btnPos.Y+btnSize.Height-2; y += 4 {
			clicked := false
			u.exportBtn.OnTapped = func() { clicked = true }
			test.TapCanvas(c, fyne.NewPos(x, y))
			if clicked {
				hits++
			} else {
				misses++
			}
		}
	}
	t.Logf("Export button WITH TOOLTIP hits: %d, misses: %d (total: %d)", hits, misses, hits+misses)
	if misses > 0 || hits == 0 {
		t.Fatalf("Expected 100%% hits on export button, got hits=%d misses=%d", hits, misses)
	}
}

