package gifdec

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"
)

// Test palette. Index 0 is the transparent entry.
var (
	cTransparent = color.RGBA{0, 0, 0, 0}
	cRed         = color.RGBA{255, 0, 0, 255}
	cGreen       = color.RGBA{0, 255, 0, 255}
	cBlue        = color.RGBA{0, 0, 255, 255}
)

var testPalette = color.Palette{cTransparent, cRed, cGreen, cBlue}

// encode builds an animated GIF with stdlib gif.EncodeAll.
func encode(t *testing.T, w, h int, frames []*image.Paletted, delays []int, disposal []byte, loopCount int) []byte {
	t.Helper()
	g := &gif.GIF{
		Image:     frames,
		Delay:     delays,
		Disposal:  disposal,
		LoopCount: loopCount,
		Config: image.Config{
			ColorModel: testPalette,
			Width:      w,
			Height:     h,
		},
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		t.Fatalf("EncodeAll: %v", err)
	}
	return buf.Bytes()
}

// paletted builds a sub-rectangle paletted image from row-major palette indices.
func paletted(left, top, width, height int, indices []byte) *image.Paletted {
	p := image.NewPaletted(image.Rect(left, top, left+width, top+height), testPalette)
	p.Pix = indices
	return p
}

var wantColors = []color.RGBA{cTransparent, cRed, cGreen, cBlue}

// wantGrid is a row-major grid of palette indices describing the expected RGBA
// of a fully composited frame.

func assertFrame(t *testing.T, label string, f Frame, w, h int, want string) {
	t.Helper()
	if f.Image == nil {
		t.Fatalf("%s: nil image", label)
	}
	b := f.Image.Bounds()
	if b.Dx() != w || b.Dy() != h {
		t.Fatalf("%s: size %d×%d, want %d×%d", label, b.Dx(), b.Dy(), w, h)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			got := f.Image.At(x, y)
			exp := wantColors[want[y*w+x]-'0']
			if got != exp {
				t.Errorf("%s: pixel (%d,%d) = %v, want %v", label, x, y, got, exp)
			}
		}
	}
}

const (
	fullRed   = "1111111111111111"
	fullGreen = "2222222222222222"
)

func TestDecodeDisposalNoneAccumulates(t *testing.T) {
	leftRedRightGreen := []byte{
		1, 1, 2, 2,
		1, 1, 2, 2,
		1, 1, 2, 2,
		1, 1, 2, 2,
	}
	leftBlue := []byte{3, 3, 3, 3, 3, 3, 3, 3} // 2×4 sub-rect at (0,0)

	data := encode(t, 4, 4,
		[]*image.Paletted{
			paletted(0, 0, 4, 4, leftRedRightGreen),
			paletted(0, 0, 2, 4, leftBlue),
		},
		[]int{10, 10}, nil, 0)

	info, frames, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(frames))
	}
	if info.Width != 4 || info.Height != 4 || info.LoopCount != 0 {
		t.Errorf("info = %+v", info)
	}
	assertFrame(t, "frame0", frames[0], 4, 4, "1122112211221122")
	want1 := "3322" + "3322" + "3322" + "3322"
	assertFrame(t, "frame1", frames[1], 4, 4, want1)
}

func TestDecodeDisposalBackgroundClearsRect(t *testing.T) {
	green2x2 := []byte{2, 2, 2, 2}

	data := encode(t, 4, 4,
		[]*image.Paletted{
			paletted(0, 0, 4, 4, []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}),
			paletted(1, 1, 2, 2, green2x2), // disposal background
			paletted(1, 1, 2, 2, []byte{0, 0, 0, 0}),
		},
		[]int{10, 10, 10}, []byte{0, gif.DisposalBackground, 0}, 0)

	_, frames, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	assertFrame(t, "frame0", frames[0], 4, 4, fullRed)

	want1 := "1111" + "1221" + "1221" + "1111"
	assertFrame(t, "frame1", frames[1], 4, 4, want1)

	// After background disposal the rect is transparent; the third frame draws
	// only transparent pixels over it, so the hole stays.
	want2 := "1111" + "1001" + "1001" + "1111"
	assertFrame(t, "frame2", frames[2], 4, 4, want2)
}

func TestDecodeDisposalPreviousRestoresRect(t *testing.T) {
	greenTopHalf := []byte{2, 2, 2, 2, 2, 2, 2, 2} // 4×2 at (0,0)

	data := encode(t, 4, 4,
		[]*image.Paletted{
			paletted(0, 0, 4, 4, []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}),
			paletted(0, 0, 4, 2, greenTopHalf), // disposal previous
			paletted(0, 0, 4, 4, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}),
		},
		[]int{10, 10, 10}, []byte{0, gif.DisposalPrevious, 0}, 0)

	_, frames, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	assertFrame(t, "frame0", frames[0], 4, 4, fullRed)

	want1 := "2222" + "2222" + "1111" + "1111"
	assertFrame(t, "frame1", frames[1], 4, 4, want1)

	// Frame 2 paints nothing; the canvas was restored to the pre-frame-1 state.
	assertFrame(t, "frame2", frames[2], 4, 4, fullRed)
}

func TestDecodeDisposalPreviousOnFirstFrame(t *testing.T) {
	leftBlue := []byte{3, 3, 3, 3, 3, 3, 3, 3} // 2×4 at (0,0)

	data := encode(t, 4, 4,
		[]*image.Paletted{
			paletted(0, 0, 4, 4, []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}),
			paletted(0, 0, 2, 4, leftBlue),
		},
		[]int{10, 10}, []byte{gif.DisposalPrevious, 0}, 0)

	_, frames, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	assertFrame(t, "frame0", frames[0], 4, 4, fullRed)

	// The canvas was restored to fully transparent before frame 1 was drawn.
	want1 := "3300" + "3300" + "3300" + "3300"
	assertFrame(t, "frame1", frames[1], 4, 4, want1)
}

func TestDecodeTransparencyKeepsUnderlyingPixel(t *testing.T) {
	// 2×4 block at (1,0): first column transparent, second column red.
	mixed := []byte{0, 1, 0, 1, 0, 1, 0, 1}

	data := encode(t, 4, 4,
		[]*image.Paletted{
			paletted(0, 0, 4, 4, []byte{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}),
			paletted(1, 0, 2, 4, mixed),
		},
		[]int{10, 10}, nil, 0)

	_, frames, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	assertFrame(t, "frame0", frames[0], 4, 4, fullGreen)

	want1 := "2212" + "2212" + "2212" + "2212"
	assertFrame(t, "frame1", frames[1], 4, 4, want1)
}

func TestDecodeFramesAreIndependentCopies(t *testing.T) {
	leftRedRightGreen := []byte{
		1, 1, 2, 2,
		1, 1, 2, 2,
		1, 1, 2, 2,
		1, 1, 2, 2,
	}
	leftBlue := []byte{3, 3, 3, 3, 3, 3, 3, 3} // 2×4 at (0,0)

	data := encode(t, 4, 4,
		[]*image.Paletted{
			paletted(0, 0, 4, 4, leftRedRightGreen),
			paletted(0, 0, 2, 4, leftBlue),
		},
		[]int{10, 10}, nil, 0)

	_, frames, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	before := frames[1].Image.RGBAAt(0, 0)
	frames[0].Image.Pix[0] = 99
	if frames[1].Image.RGBAAt(0, 0) != before {
		t.Errorf("frames share a backing array")
	}
	if &frames[0].Image.Pix[0] == &frames[1].Image.Pix[0] {
		t.Errorf("frames share a backing array")
	}
}

func TestDecodeDelay(t *testing.T) {
	px := []byte{1, 1, 1, 1}
	data := encode(t, 4, 1,
		[]*image.Paletted{
			paletted(0, 0, 4, 1, px), paletted(0, 0, 4, 1, px),
			paletted(0, 0, 4, 1, px), paletted(0, 0, 4, 1, px),
			paletted(0, 0, 4, 1, px),
		},
		[]int{0, 1, 10, 25, 2}, nil, 0)

	_, frames, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := []int{100, 100, 100, 250, 20}
	for i, w := range want {
		if frames[i].DelayMs != w {
			t.Errorf("frame %d delay = %d, want %d", i, frames[i].DelayMs, w)
		}
		if frames[i].Index != i {
			t.Errorf("frame %d has Index %d", i, frames[i].Index)
		}
	}
}

func TestDecodeLoopCount(t *testing.T) {
	px := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	for _, loop := range []int{-1, 0, 5} {
		data := encode(t, 4, 4, []*image.Paletted{paletted(0, 0, 4, 4, px), paletted(0, 0, 4, 4, px)}, []int{10, 10}, nil, loop)
		info, _, err := Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("Decode(loop=%d): %v", loop, err)
		}
		if info.LoopCount != loop {
			t.Errorf("LoopCount = %d, want %d", info.LoopCount, loop)
		}
	}
}

func TestDecodeEstimateBytesTooLarge(t *testing.T) {
	if got := EstimateBytes(10, 20, 3); got != 10*20*4*3 {
		t.Errorf("EstimateBytes = %d", got)
	}

	old := MaxBytes
	MaxBytes = 4*4*4*2 - 1 // just under two 4×4 frames
	defer func() { MaxBytes = old }()

	px := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	data := encode(t, 4, 4,
		[]*image.Paletted{paletted(0, 0, 4, 4, px), paletted(0, 0, 4, 4, px)},
		[]int{10, 10}, nil, 0)

	if _, _, err := Decode(bytes.NewReader(data)); err != ErrTooLarge {
		t.Errorf("err = %v, want ErrTooLarge", err)
	}
}

func TestDecodeInvalidInput(t *testing.T) {
	px := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	data := encode(t, 4, 4, []*image.Paletted{paletted(0, 0, 4, 4, px)}, []int{10}, nil, 0)

	if _, _, err := Decode(bytes.NewReader([]byte("not a gif at all"))); err == nil {
		t.Error("expected error for non-GIF input")
	}
	if _, _, err := Decode(bytes.NewReader(data[:len(data)/2])); err == nil {
		t.Error("expected error for truncated GIF")
	}
}
