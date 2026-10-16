// Package export writes selected frames as PNG files.
package export

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mamorett/qgif/internal/gifdec"
)

// Options controls where and how frames are written.
type Options struct {
	Dir       string
	BaseName  string // GIF file name without extension, sanitised by Sanitise
	Overwrite bool
}

// Result summarises an export run.
type Result struct {
	Written int
	Failed  []error
}

const illegalChars = `<>:"/\|?*`

// Sanitise replaces characters that are illegal in file names (and control
// characters) with underscores. An empty result becomes "frames".
func Sanitise(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(illegalChars, r) {
			b.WriteByte('_')
		} else {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "frames"
	}
	return out
}

// padding returns the digit width for frame numbers: max(4, digits(total)).
func padding(total int) int {
	d := len(strconv.Itoa(total))
	if d < 4 {
		return 4
	}
	return d
}

// fileName builds <BaseName>_frame_<NNNN>.png for the 0-based index.
func fileName(opts Options, index, total int) string {
	num := strconv.Itoa(index + 1)
	pad := padding(total)
	if len(num) < pad {
		num = strings.Repeat("0", pad-len(num)) + num
	}
	return filepath.Join(opts.Dir, Sanitise(opts.BaseName)+"_frame_"+num+".png")
}

// Existing returns the target paths of the given indices that already exist.
func Existing(opts Options, indices []int, total int) []string {
	var out []string
	for _, i := range indices {
		p := fileName(opts, i, total)
		if _, err := os.Stat(p); err == nil {
			out = append(out, p)
		}
	}
	return out
}

// Run writes each selected frame as a PNG. progress(done, total) is called
// after each file (skips included). Existing files are skipped unless
// Overwrite; skipped files count as neither written nor failed.
func Run(frames []gifdec.Frame, indices []int, opts Options, progress func(done, total int)) Result {
	var res Result
	total := len(frames)

	for n, i := range indices {
		if i < 0 || i >= len(frames) {
			res.Failed = append(res.Failed,
				fmt.Errorf("frame %d out of range (0..%d)", i, len(frames)-1))
			report(progress, n+1, len(indices))
			continue
		}
		target := fileName(opts, i, total)
		if !opts.Overwrite {
			if _, err := os.Stat(target); err == nil {
				report(progress, n+1, len(indices))
				continue
			}
		}
		if err := writeAtomic(target, frames[i].Image); err != nil {
			res.Failed = append(res.Failed, fmt.Errorf("%s: %w", filepath.Base(target), err))
		} else {
			res.Written++
		}
		report(progress, n+1, len(indices))
	}
	return res
}

func report(progress func(done, total int), done, total int) {
	if progress != nil {
		progress(done, total)
	}
}

// writeAtomic encodes to a temp file in the same folder, then renames.
func writeAtomic(target string, img *image.RGBA) (err error) {
	tmp := target + ".tmp"
	defer func() {
		if err != nil {
			os.Remove(tmp)
		}
	}()

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err = png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}
