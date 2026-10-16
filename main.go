// qGif — pick frames from an animated GIF and export them as PNG files.
package main

import (
	_ "embed"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/mamorett/qgif/internal/ui"
)

// The Linux build is a plain `go build`, so unlike the macOS bundle it carries
// no icon metadata; without this the window falls back to Fyne's stock logo.
//
//go:embed icon.png
var appIcon []byte

func main() {
	a := app.NewWithID("io.github.qgif.qgif")
	w := a.NewWindow("qGif")
	w.SetIcon(&fyne.StaticResource{StaticName: "qgif.png", StaticContent: appIcon})

	u := ui.New(a, w)

	if len(os.Args) > 1 {
		u.OpenPath(os.Args[1])
	}

	w.Resize(fyne.NewSize(1100, 750))
	w.CenterOnScreen()
	w.ShowAndRun()
}
