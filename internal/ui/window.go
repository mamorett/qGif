// Package ui wires the Fyne widgets to the pure-Go core packages.
package ui

import (
	_ "embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mamorett/qgif/internal/export"
	"github.com/mamorett/qgif/internal/gifdec"
	"github.com/mamorett/qgif/internal/selection"
	"github.com/mamorett/qgif/internal/thumbs"
)

var dbgOn = os.Getenv("QGIF_DEBUG") != ""

// Version is shown in the About dialog and the footer.
const Version = "0.1.0"

const lastExportDirKey = "lastExportDir"

// doubleClickWindow is the maximum gap between two clicks on the same card
// for the second one to count as a double click (select).
const doubleClickWindow = 300 * time.Millisecond

//go:embed logo.png
var logoBytes []byte

// App owns the main window state and widgets.
type App struct {
	a fyne.App
	w fyne.Window

	grid      *widget.GridWrap
	left      *fyne.Container
	empty     *fyne.Container
	status    *widget.Label
	statusDot *canvas.Circle
	split     *container.Split
	preview   *Preview

	openBtn         *hoverButton
	cleanBtn        *hoverButton
	allBtn          *hoverButton
	exportBtn       *hoverButton
	openItem        *fyne.MenuItem
	exportItem      *fyne.MenuItem
	exportOtherItem *fyne.MenuItem
	fileMenu        *fyne.Menu
	tooltipLayer    *fyne.Container

	loading bool


	frames      []gifdec.Frame
	thumbImgs   []*image.RGBA
	delays      []int
	sel         *selection.Set
	info        gifdec.Info
	fileName    string
	baseName    string
	lastGifPath string

	// current is the single-clicked "viewing" frame; exported when nothing
	// is selected. lastTap* implement double-click detection.
	current      int
	lastTapIndex int
	lastTapTime  time.Time
}

// New builds the main window content, header, footer, menus and shortcuts.
func New(a fyne.App, w fyne.Window) *App {
	a.Settings().SetTheme(NewQgifTheme())

	u := &App{a: a, w: w, current: -1, lastTapIndex: -1}
	u.tooltipLayer = container.NewWithoutLayout()

	u.preview = NewPreview()

	u.grid = widget.NewGridWrap(
		func() int { return len(u.thumbImgs) },
		func() fyne.CanvasObject { return newFrameCard(u.onCardTap) },
		func(id widget.GridWrapItemID, o fyne.CanvasObject) {
			o.(*FrameCard).Set(id, u.thumbImgs[id], u.sel.IsSelected(id), u.current == id, u.delays[id])
		},
	)
	u.grid.Hide()

	dz := NewDropZone("DRAG & DROP A GIF HERE — or click Open (Ctrl/⌘+O)")
	openBig := newHoverButton(theme.FolderOpenIcon(), "Open a GIF (Ctrl/⌘+O)", u.openDialog, w, u.tooltipLayer)
	openBig.Importance = widget.HighImportance
	u.empty = container.NewCenter(container.NewVBox(
		container.NewCenter(dz),
		container.NewCenter(openBig),
	))
	u.left = container.NewStack(u.grid, u.empty)

	// Status (goKomfy style: dot + monospace label, right-aligned; About and
	// italic version on the left).
	u.status = widget.NewLabel("No GIF loaded")
	u.status.TextStyle = fyne.TextStyle{Monospace: true}
	u.statusDot = canvas.NewCircle(theme.Color(theme.ColorNameSuccess))
	u.statusDot.Resize(fyne.NewSize(8, 8))
	u.statusDot.Hide()

	aboutBtn := newHoverButton(theme.InfoIcon(), "About qGif", u.about, w, u.tooltipLayer)
	version := widget.NewLabel("v" + Version)
	version.TextStyle = fyne.TextStyle{Italic: true}

	footer := container.NewBorder(nil, nil,
		container.NewHBox(aboutBtn, version),
		container.NewHBox(container.NewCenter(u.statusDot), u.status),
		layout.NewSpacer())

	// Header (goKomfy style: logo + bold title + spacer + icon buttons).
	logo := widget.NewIcon(fyne.NewStaticResource("logo.png", logoBytes))
	title := widget.NewLabelWithStyle("qGif", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	u.openBtn = newHoverButton(theme.FolderOpenIcon(), "Open a GIF (Ctrl/⌘+O)", u.openDialog, w, u.tooltipLayer)
	u.cleanBtn = newHoverButton(theme.DeleteIcon(), "Unload the current GIF (Ctrl/⌘+Shift+W)", u.unload, w, u.tooltipLayer)
	u.cleanBtn.Disable()
	u.allBtn = newHoverButton(theme.CheckButtonCheckedIcon(), "Select all frames (Ctrl/⌘+A)", u.selectAll, w, u.tooltipLayer)
	noneBtn := newHoverButton(theme.CheckButtonIcon(), "Clear selection (Ctrl/⌘+Shift+A)", u.selectNone, w, u.tooltipLayer)
	invertBtn := newHoverButton(theme.ViewRefreshIcon(), "Invert selection (Ctrl/⌘+I)", u.invert, w, u.tooltipLayer)
	u.exportBtn = newHoverButtonWithText("Save", theme.DocumentSaveIcon(),
		"Save selected frames — or the viewed frame — as PNGs into the GIF's folder (Ctrl/⌘+E)", u.exportFlow, w, u.tooltipLayer)
	u.exportBtn.Importance = widget.HighImportance
	u.exportBtn.Disable()

	header := container.NewHBox(
		logo, title,
		layout.NewSpacer(),
		u.openBtn, u.cleanBtn,
		widget.NewSeparator(),
		u.allBtn, noneBtn, invertBtn,
		widget.NewSeparator(),
		u.exportBtn,
	)

	u.split = container.NewHSplit(u.left, u.preview.Container)
	u.split.Offset = 0.72

	content := container.NewBorder(
		container.NewPadded(header),
		container.NewPadded(footer),
		nil, nil,
		container.NewPadded(u.split),
	)
	root := container.NewStack(content, u.tooltipLayer)
	w.SetContent(root)


	u.buildMenus()

	w.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		for _, uri := range uris {
			if strings.EqualFold(filepath.Ext(uri.Path()), ".gif") {
				if !u.grid.Visible() {
					dz.Flash()
				}
				u.startLoad(uri.Path())
				return
			}
		}
	})
	return u
}

// OpenPath loads a GIF from the local disk (used by the CLI argument).
func (u *App) OpenPath(path string) { u.startLoad(path) }

// ---------------------------------------------------------------- selection

// onCardTap implements the click semantics:
//   - single click  → display the frame (preview + Save exports it)
//   - double click  → toggle selection (Save exports the selection)
//   - Shift+click   → select the range from the anchor
func (u *App) onCardTap(i int, shift bool) {
	if dbgOn {
		fmt.Fprintf(os.Stderr, "DBG onCardTap i=%d shift=%v last=%d dt=%v\n", i, shift, u.lastTapIndex, time.Since(u.lastTapTime))
	}
	if u.sel == nil {
		return
	}
	if shift {
		u.sel.SelectRange(i)
		u.current = i
		u.lastTapIndex = -1
	} else if i == u.lastTapIndex && time.Since(u.lastTapTime) < doubleClickWindow {
		u.sel.Toggle(i)
		u.current = i
		u.lastTapIndex = -1 // next click starts fresh
	} else {
		// Single click: view this frame and make it the range anchor so a
		// following Shift+click selects the range from here.
		u.current = i
		u.sel.SetAnchor(i)
		u.lastTapIndex, u.lastTapTime = i, time.Now()
	}
	u.preview.ShowFrame(u.frames[i], len(u.frames))
	u.selectionChanged()
}

func (u *App) selectAll()  { u.applySelection(u.sel.SelectAll) }
func (u *App) selectNone() { u.applySelection(u.sel.SelectNone) }
func (u *App) invert()     { u.applySelection(u.sel.Invert) }

func (u *App) applySelection(op func()) {
	if u.sel == nil || u.sel.Len() == 0 {
		return
	}
	op()
	u.selectionChanged()
}

func (u *App) selectionChanged() {
	u.grid.Refresh()
	u.updateStatus()
	u.updateExportEnabled()
}

func (u *App) updateExportEnabled() {
	enabled := u.sel != nil && (u.sel.Count() > 0 || u.current >= 0)
	if enabled {
		u.exportBtn.Enable()
		u.exportItem.Disabled = false
		u.exportOtherItem.Disabled = false
	} else {
		u.exportBtn.Disable()
		u.exportItem.Disabled = true
		u.exportOtherItem.Disabled = true
	}
	u.fileMenu.Refresh()
}

func (u *App) updateStatus() {
	if u.sel == nil || len(u.frames) == 0 {
		u.status.SetText("No GIF loaded")
		return
	}
	base := fmt.Sprintf("%s · %d×%d · %d frames", u.fileName, u.info.Width, u.info.Height, len(u.frames))
	switch {
	case u.sel.Count() > 0:
		u.status.SetText(fmt.Sprintf("%s · Selected: %d", base, u.sel.Count()))
	case u.current >= 0:
		u.status.SetText(fmt.Sprintf("%s · Viewing #%04d — double-click to select", base, u.current+1))
	default:
		u.status.SetText(fmt.Sprintf("%s · Click a frame", base))
	}
}

// unload clears the loaded GIF and returns the app to its empty state.
func (u *App) unload() {
	if len(u.frames) == 0 && u.lastGifPath == "" {
		return
	}
	u.frames, u.thumbImgs, u.delays = nil, nil, nil
	u.sel = nil
	u.info = gifdec.Info{}
	u.fileName, u.baseName, u.lastGifPath = "", "", ""
	u.current, u.lastTapIndex, u.lastTapTime = -1, -1, time.Time{}
	u.preview.Clear()
	u.grid.Hide()
	u.empty.Show()
	u.status.SetText("No GIF loaded")
	u.statusDot.Hide()
	u.w.SetTitle("qGif")
	u.cleanBtn.Disable()
	u.updateExportEnabled()
}

func (u *App) about() {
	logo := canvas.NewImageFromResource(fyne.NewStaticResource("logo.png", logoBytes))
	logo.FillMode = canvas.ImageFillOriginal
	logo.SetMinSize(fyne.NewSize(96, 96))

	title := widget.NewLabelWithStyle("qGif", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	version := widget.NewLabelWithStyle("v"+Version, fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	desc := widget.NewLabelWithStyle("Pick GIF frames and export them as PNG.", fyne.TextAlignCenter, fyne.TextStyle{})
	desc.Wrapping = fyne.TextWrapWord

	var dlg *dialog.CustomDialog
	okBtn := widget.NewButton("OK", func() { dlg.Hide() })
	okBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		container.NewCenter(logo),
		container.NewCenter(title),
		container.NewCenter(version),
		container.NewCenter(desc),
		layout.NewSpacer(),
		container.NewCenter(okBtn),
	)

	dlg = dialog.NewCustomWithoutButtons("About qGif", container.NewPadded(content), u.w)
	dlg.Resize(fyne.NewSize(340, 440))
	dlg.Show()
}

// ---------------------------------------------------------------- open flow

func (u *App) openDialog() {
	if runtime.GOOS == "linux" {
		start := "."
		if u.lastGifPath != "" {
			start = filepath.Dir(u.lastGifPath)
		} else if h, err := os.UserHomeDir(); err == nil {
			start = h
		}
		go func() {
			path, chosen, available, err := nativeFilePick("Open a GIF", start, "*.gif *.GIF")
			if !available {
				fyne.Do(u.fyneOpenFlow)
				return
			}
			if err != nil || !chosen {
				return // cancelled or picker error
			}
			fyne.Do(func() { u.startLoad(path) })
		}()
		return
	}
	u.fyneOpenFlow()
}

func (u *App) fyneOpenFlow() {
	d := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, u.w)
			return
		}
		if rc == nil {
			return
		}
		path := rc.URI().Path()
		rc.Close()
		u.startLoad(path)
	}, u.w)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".gif", ".GIF"}))
	d.Resize(fyne.NewSize(700, 500))
	d.Show()
}

// startLoad decodes and thumbnails path in the background.
func (u *App) startLoad(path string) {
	if u.loading {
		return
	}
	u.loading = true
	u.openBtn.Disable()
	u.openItem.Disabled = true
	u.fileMenu.Refresh()
	u.setStatusDot(theme.Color(theme.ColorNameWarning))

	dlg := dialog.NewCustomWithoutButtons("Loading…", widget.NewProgressBarInfinite(), u.w)
	dlg.Resize(fyne.NewSize(300, 120))
	dlg.Show()

	go func() {
		f, err := os.Open(path)
		if err != nil {
			u.loadFailed(dlg, err)
			return
		}
		defer f.Close()

		info, frames, err := gifdec.Decode(f)
		if err != nil {
			if errors.Is(err, gifdec.ErrTooLarge) {
				err = fmt.Errorf("This GIF is too large to open (%d×%d, %d frames, ~%s)",
					info.Width, info.Height, info.Frames, humanBytes(gifdec.EstimateBytes(info.Width, info.Height, info.Frames)))
			} else {
				err = fmt.Errorf("Cannot decode GIF: %w", err)
			}
			u.loadFailed(dlg, err)
			return
		}

		// Build thumbnails with a CPU-sized worker pool.
		n := runtime.NumCPU()
		if n < 1 {
			n = 1
		}
		tp := make([]*image.RGBA, len(frames))
		var wg sync.WaitGroup
		for worker := 0; worker < n; worker++ {
			wg.Add(1)
			go func(start int) {
				defer wg.Done()
				for i := start; i < len(frames); i += n {
					tp[i] = thumbs.Make(frames[i].Image, thumbMaxSide)
				}
			}(worker)
		}
		wg.Wait()

		delays := make([]int, len(frames))
		for i := range frames {
			delays[i] = frames[i].DelayMs
		}

		fyne.Do(func() {
			dlg.Hide()
			u.loading = false
			u.openBtn.Enable()
			u.openItem.Disabled = false
			u.fileMenu.Refresh()

			u.frames = frames
			u.thumbImgs = tp
			u.delays = delays
			u.sel = selection.New(len(frames))
			u.info = info
			u.fileName = filepath.Base(path)
			u.baseName = strings.TrimSuffix(u.fileName, filepath.Ext(u.fileName))
			u.lastGifPath = path
			u.current = 0
			u.lastTapIndex = -1
			u.cleanBtn.Enable()

			u.empty.Hide()
			u.grid.Show()
			u.grid.Refresh()
			u.grid.ScrollToTop()
			u.preview.ShowFrame(u.frames[0], len(u.frames))
			u.setStatusDot(theme.Color(theme.ColorNameSuccess))
			u.updateStatus()
			u.w.SetTitle("qGif — " + u.fileName)
			u.updateExportEnabled()
		})
	}()
}

func (u *App) loadFailed(dlg *dialog.CustomDialog, err error) {
	fyne.Do(func() {
		dlg.Hide()
		u.loading = false
		u.openBtn.Enable()
		u.openItem.Disabled = false
		u.fileMenu.Refresh()
		u.setStatusDot(theme.Color(theme.ColorNameError))
		dialog.ShowError(err, u.w)
	})
}

func (u *App) setStatusDot(c color.Color) {
	u.statusDot.FillColor = c
	u.statusDot.Show()
	u.statusDot.Refresh()
}

func humanBytes(b int64) string {
	const mib = 1 << 20
	const gib = 1 << 30
	switch {
	case b >= gib:
		return fmt.Sprintf("%.1f GiB", float64(b)/gib)
	case b >= mib:
		return fmt.Sprintf("%.0f MiB", float64(b)/mib)
	default:
		return fmt.Sprintf("%.0f KiB", float64(b)/(1<<10))
	}
}

// --------------------------------------------------------------- export flow

// exportFlow saves the selected frames, or the currently viewed (single
// clicked) frame when nothing is selected.
// exportIndices returns what Save exports: the selection, or the viewed
// frame when nothing is selected.
func (u *App) exportIndices() []int {
	if u.sel == nil || len(u.frames) == 0 {
		return nil
	}
	if indices := u.sel.Indices(); len(indices) > 0 {
		return indices
	}
	if u.current >= 0 {
		return []int{u.current}
	}
	return nil
}

// exportFlow is the primary Save action: writes the selected frames (or the
// viewed frame) straight into the GIF's folder with auto-composed names.
// No filename is ever asked; existing files trigger an overwrite confirm.
func (u *App) exportFlow() {
	indices := u.exportIndices()
	if len(indices) == 0 {
		return
	}
	u.confirmAndExport(indices, u.exportTargetDir())
}

// exportToOtherFolderFlow lets the user override the destination folder via
// the native picker (kdialog/zenity) or Fyne's dialog as last resort.
func (u *App) exportToOtherFolderFlow() {
	indices := u.exportIndices()
	if len(indices) == 0 {
		return
	}

	if runtime.GOOS == "linux" {
		start := u.exportTargetDir()
		go func() {
			dir, chosen, available, _ := nativeFolderPick("Choose folder to save frames into", start)
			if !available {
				fyne.Do(func() { u.fyneFolderFlow(indices, start) })
				return
			}
			if !chosen {
				return // cancelled
			}
			fyne.Do(func() { u.confirmAndExport(indices, dir) })
		}()
		return
	}

	u.fyneFolderFlow(indices, u.exportTargetDir())
}

// exportTargetDir is the folder preselected for the export dialog: the
// directory of the loaded GIF, falling back to the last used folder.
func (u *App) exportTargetDir() string {
	if u.lastGifPath != "" {
		if d := filepath.Dir(u.lastGifPath); d != "" {
			return d
		}
	}
	if last := u.a.Preferences().StringWithFallback(lastExportDirKey, ""); last != "" {
		if fi, err := os.Stat(last); err == nil && fi.IsDir() {
			return last
		}
	}
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "."
}

func (u *App) fyneFolderFlow(indices []int, startDir string) {
	d := dialog.NewFolderOpen(func(loc fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, u.w)
			return
		}
		if loc == nil {
			return // cancelled
		}
		u.confirmAndExport(indices, loc.Path())
	}, u.w)

	if fi, err := os.Stat(startDir); err == nil && fi.IsDir() {
		if lu, err := storage.ListerForURI(storage.NewFileURI(startDir)); err == nil {
			d.SetLocation(lu)
		}
	}
	d.Resize(fyne.NewSize(700, 500))
	d.Show()
}

// confirmAndExport asks about existing files, then writes.
func (u *App) confirmAndExport(indices []int, dir string) {
	u.a.Preferences().SetString(lastExportDirKey, dir)
	opts := export.Options{Dir: dir, BaseName: u.baseName}
	if existing := export.Existing(opts, indices, len(u.frames)); len(existing) > 0 {
		confirm := dialog.NewConfirm("Files exist",
			fmt.Sprintf("%d files already exist. Overwrite them?", len(existing)),
			func(yes bool) {
				opts.Overwrite = yes
				u.runExport(indices, opts)
			}, u.w)
		confirm.Show()
		return
	}
	u.runExport(indices, opts)
}

func (u *App) runExport(indices []int, opts export.Options) {
	bar := widget.NewProgressBar()
	bar.Min = 0
	bar.Max = float64(len(indices))
	dlg := dialog.NewCustomWithoutButtons("Exporting…", bar, u.w)
	dlg.Resize(fyne.NewSize(320, 120))
	dlg.Show()

	frames := u.frames // snapshot; independent of later loads
	go func() {
		res := export.Run(frames, indices, opts, func(done, total int) {
			fyne.Do(func() { bar.SetValue(float64(done)) })
		})
		fyne.Do(func() {
			dlg.Hide()
			msg := fmt.Sprintf("Exported %d frames to %s", res.Written, opts.Dir)
			if n := len(res.Failed); n > 0 {
				msg += fmt.Sprintf("\n%d failed:\n", n)
				for i, err := range res.Failed {
					if i == 5 {
						msg += "…"
						break
					}
					msg += err.Error() + "\n"
				}
			}
			dialog.ShowInformation("Export complete", strings.TrimRight(msg, "\n"), u.w)
		})
	}()
}

// ---------------------------------------------------------------- menus etc

func (u *App) buildMenus() {
	mod := fyne.KeyModifierShortcutDefault

	u.openItem = fyne.NewMenuItem("Open…", u.openDialog)
	u.openItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: mod}

	unloadItem := fyne.NewMenuItem("Unload GIF", u.unload)
	unloadItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: mod | fyne.KeyModifierShift}

	u.exportItem = fyne.NewMenuItem("Export to GIF Folder", u.exportFlow)
	u.exportItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: mod}
	u.exportItem.Disabled = true
	exportOther := fyne.NewMenuItem("Export to Another Folder…", u.exportToOtherFolderFlow)
	exportOther.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: mod | fyne.KeyModifierShift}
	exportOther.Disabled = true
	u.exportOtherItem = exportOther
	u.fileMenu = fyne.NewMenu("File", u.openItem, unloadItem, u.exportItem, exportOther)

	all := fyne.NewMenuItem("Select All", u.selectAll)
	all.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyA, Modifier: mod}
	none := fyne.NewMenuItem("Select None", u.selectNone)
	none.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyA, Modifier: mod | fyne.KeyModifierShift}
	inv := fyne.NewMenuItem("Invert Selection", u.invert)
	inv.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyI, Modifier: mod}
	editMenu := fyne.NewMenu("Edit", all, none, inv)

	helpMenu := fyne.NewMenu("Help", fyne.NewMenuItem("About", u.about))

	u.w.SetMainMenu(fyne.NewMainMenu(u.fileMenu, editMenu, helpMenu))

	canvas := u.w.Canvas()
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: mod | fyne.KeyModifierShift},
		func(fyne.Shortcut) { u.unload() })
	canvas.AddShortcut(exportOther.Shortcut, func(fyne.Shortcut) { u.exportToOtherFolderFlow() })
	canvas.AddShortcut(all.Shortcut, func(fyne.Shortcut) { u.selectAll() })
	canvas.AddShortcut(none.Shortcut, func(fyne.Shortcut) { u.selectNone() })
	canvas.AddShortcut(inv.Shortcut, func(fyne.Shortcut) { u.invert() })
}
