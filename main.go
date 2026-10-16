// qGif — pick frames from an animated GIF and export them as PNG files.
package main

import (
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/mamorett/qgif/internal/ui"
)

func main() {
	a := app.NewWithID("io.github.qgif.qgif")
	w := a.NewWindow("qGif")

	u := ui.New(a, w)

	if len(os.Args) > 1 {
		u.OpenPath(os.Args[1])
	}

	w.Resize(fyne.NewSize(1100, 750))
	w.CenterOnScreen()
	w.ShowAndRun()
}
