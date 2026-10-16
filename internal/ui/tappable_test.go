package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestHeaderButtonIsTappable(t *testing.T) {
	u := newLoadedApp(t, 3)
	if _, ok := interface{}(u.allBtn).(fyne.Tappable); !ok {
		t.Fatal("hoverButton does NOT implement fyne.Tappable")
	}
	t.Log("implements Tappable")
	test.Tap(u.allBtn)
	if u.sel.Count() != 3 {
		t.Fatalf("Tap did not trigger action: count=%d", u.sel.Count())
	}
}
