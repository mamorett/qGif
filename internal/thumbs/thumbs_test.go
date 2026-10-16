package thumbs

import (
	"image"
	"image/color"
	"testing"
)

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestMakeKeepsAspectRatio(t *testing.T) {
	src := solid(800, 200, color.RGBA{255, 0, 0, 255})
	th := Make(src, 320)
	b := th.Bounds()
	if b.Dx() != 320 {
		t.Fatalf("width = %d, want 320", b.Dx())
	}
	if b.Dy() != 80 {
		t.Fatalf("height = %d, want 80", b.Dy())
	}
	// Uniform colour must survive the scale.
	if got := th.RGBAAt(10, 10); got != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("pixel = %v, want solid red", got)
	}
}

func TestMakeTallImage(t *testing.T) {
	src := solid(100, 400, color.RGBA{0, 255, 0, 255})
	th := Make(src, 320)
	b := th.Bounds()
	if b.Dy() != 320 || b.Dx() != 80 {
		t.Fatalf("size = %v, want 80×320", b.Dx())
	}
}

func TestMakeNoUpscale(t *testing.T) {
	src := solid(37, 12, color.RGBA{0, 0, 255, 255})
	th := Make(src, 320)
	b := th.Bounds()
	if b.Dx() != 37 || b.Dy() != 12 {
		t.Fatalf("size = %d×%d, want 37×12 (no upscale)", b.Dx(), b.Dy())
	}
	if got := th.RGBAAt(5, 5); got != (color.RGBA{0, 0, 255, 255}) {
		t.Fatalf("pixel = %v, want solid blue", got)
	}
}

func TestMakeReturnsCopy(t *testing.T) {
	src := solid(500, 10, color.RGBA{1, 2, 3, 255})
	th := Make(src, 320)
	orig := th.RGBAAt(0, 0)
	src.Pix[0] = 200
	if th.RGBAAt(0, 0) != orig {
		t.Fatal("thumbnail shares memory with source")
	}
}
