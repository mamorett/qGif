// Package gifdec decodes animated GIFs into fully composited RGBA frames.
package gifdec

import (
	"errors"
	"image"
	"image/draw"
	"image/gif"
	"io"
)

// ErrTooLarge is returned when a GIF's decompressed size exceeds the memory guard.
var ErrTooLarge = errors.New("gif too large")

// MaxBytes is the memory guard: frames larger than this are refused (§4.4).
// It is a var only so tests can lower it without allocating gigabytes.
var MaxBytes int64 = 2 << 30 // 2 GiB

// Frame is one fully composited animation frame.
type Frame struct {
	Index   int         // 0-based
	Image   *image.RGBA // full canvas, independent copy
	DelayMs int         // effective delay in ms
}

// Info describes the decoded animation.
type Info struct {
	Width, Height int
	LoopCount     int // from gif.GIF.LoopCount: 0 = infinite, -1 = play once, n = n extra loops
	Frames        int // total number of frames in the file
}

// EstimateBytes returns the approximate decompressed size of frames frames of w×h.
func EstimateBytes(w, h, frames int) int64 {
	return int64(w) * int64(h) * 4 * int64(frames)
}

// Decode reads an animated GIF and returns every frame fully composited.
func Decode(r io.Reader) (Info, []Frame, error) {
	g, err := gif.DecodeAll(r)
	if err != nil {
		return Info{}, nil, err
	}

	w, h := g.Config.Width, g.Config.Height
	if w == 0 || h == 0 {
		// Some encoders write a 0×0 logical screen; fall back to the union of frame bounds.
		var rect image.Rectangle
		for _, img := range g.Image {
			rect = rect.Union(img.Bounds())
		}
		w, h = rect.Dx(), rect.Dy()
		if w == 0 || h == 0 {
			return Info{}, nil, errors.New("gif has no usable dimensions")
		}
	}

	if EstimateBytes(w, h, len(g.Image)) > MaxBytes {
		return Info{Width: w, Height: h, LoopCount: g.LoopCount, Frames: len(g.Image)}, nil, ErrTooLarge
	}

	info := Info{Width: w, Height: h, LoopCount: g.LoopCount, Frames: len(g.Image)}
	canvas := image.NewRGBA(image.Rect(0, 0, w, h))
	frames := make([]Frame, 0, len(g.Image))

	for i, src := range g.Image {
		disposal := byte(0)
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}
		rect := src.Bounds().Intersect(canvas.Bounds())

		var saved *image.RGBA
		if disposal == gif.DisposalPrevious {
			saved = cloneRect(canvas, rect)
		}

		draw.Draw(canvas, rect, src, rect.Min, draw.Over)

		delay := 0
		if i < len(g.Delay) {
			delay = g.Delay[i]
		}
		frames = append(frames, Frame{
			Index:   i,
			Image:   clone(canvas),
			DelayMs: effectiveDelay(delay),
		})

		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvas, rect, image.Transparent, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			draw.Draw(canvas, rect, saved, rect.Min, draw.Src)
		default: // 0, 1 and unknown values: keep the canvas as-is
		}
	}

	return info, frames, nil
}

// effectiveDelay converts centiseconds to ms using browser rules.
func effectiveDelay(cs int) int {
	if cs <= 1 {
		return 100
	}
	return cs * 10
}

func clone(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Rect)
	copy(dst.Pix, src.Pix)
	return dst
}

// cloneRect copies only the pixels of rect from src.
func cloneRect(src *image.RGBA, rect image.Rectangle) *image.RGBA {
	dst := image.NewRGBA(rect)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		srcOff := src.PixOffset(rect.Min.X, y)
		dstOff := dst.PixOffset(rect.Min.X, y)
		width := rect.Dx() * 4
		copy(dst.Pix[dstOff:dstOff+width], src.Pix[srcOff:srcOff+width])
	}
	return dst
}
