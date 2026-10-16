// Package thumbs builds small RGBA thumbnails from full frames.
package thumbs

import (
	"image"

	xdraw "golang.org/x/image/draw"
)

// Make returns a copy of src scaled to fit inside maxSide×maxSide, keeping
// aspect ratio. Images smaller than maxSide are returned unscaled (a copy).
func Make(src *image.RGBA, maxSide int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	dstW, dstH := w, h
	if w > maxSide || h > maxSide {
		if w >= h {
			dstW = maxSide
			dstH = maxSide * h / w
		} else {
			dstH = maxSide
			dstW = maxSide * w / h
		}
		if dstW < 1 {
			dstW = 1
		}
		if dstH < 1 {
			dstH = 1
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, b, xdraw.Src, nil)
	return dst
}
