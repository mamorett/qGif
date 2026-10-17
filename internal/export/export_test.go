package export

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mamorett/qgif/internal/gifdec"
)

func makeFrame(t *testing.T, w, h int, seed byte) gifdec.Frame {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{seed, byte(x * 16), byte(y * 16), 255})
		}
	}
	return gifdec.Frame{Image: img}
}

func TestRunWritesAndRoundTrips(t *testing.T) {
	dir := t.TempDir()
	fs := []gifdec.Frame{makeFrame(t, 6, 3, 7), makeFrame(t, 6, 3, 9)}

	var done, totalSeen int
	res := Run(fs, []int{0, 1}, Options{Dir: dir, BaseName: "cat"}, func(done_, total int) {
		done, totalSeen = done_, total
	})
	if res.Written != 2 || len(res.Failed) != 0 {
		t.Fatalf("result = %+v", res)
	}
	if done != 2 || totalSeen != 2 {
		t.Fatalf("progress = (%d,%d), want (2,2)", done, totalSeen)
	}

	for i, name := range []string{"cat_frame_0001.png", "cat_frame_0002.png"} {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatalf("%s: decode: %v", name, err)
		}
		want := fs[i].Image
		if img.Bounds() != want.Bounds() {
			t.Fatalf("%s: bounds %v vs %v", name, img.Bounds(), want.Bounds())
		}
		for y := 0; y < want.Bounds().Dy(); y++ {
			for x := 0; x < want.Bounds().Dx(); x++ {
				if img.At(x, y) != want.At(x, y) {
					t.Fatalf("%s: pixel (%d,%d) = %v, want %v", name, x, y, img.At(x, y), want.At(x, y))
				}
			}
		}
	}
}

func TestRunPadding(t *testing.T) {
	dir := t.TempDir()
	// total = 100000 → digits = 6 → padding 6.
	fs := make([]gifdec.Frame, 100000)
	fs[99999] = makeFrame(t, 1, 1, 3)
	res := Run(fs, []int{99999}, Options{Dir: dir, BaseName: "x"}, nil)
	if res.Written != 1 || len(res.Failed) != 0 {
		t.Fatalf("result = %+v", res)
	}
	if _, err := os.Stat(filepath.Join(dir, "x_frame_100000.png")); err != nil {
		t.Fatalf("expected padded name: %v", err)
	}

	// total < 10000 → padding stays 4.
	dir2 := t.TempDir()
	small := make([]gifdec.Frame, 120)
	small[4] = makeFrame(t, 1, 1, 5)
	Run(small, []int{4}, Options{Dir: dir2, BaseName: "cat"}, nil)
	if _, err := os.Stat(filepath.Join(dir2, "cat_frame_0005.png")); err != nil {
		t.Fatalf("expected 4-digit name: %v", err)
	}
}

func TestRunSkipVsOverwrite(t *testing.T) {
	dir := t.TempDir()
	fs := []gifdec.Frame{makeFrame(t, 2, 2, 1)}
	target := filepath.Join(dir, "cat_frame_0001.png")

	// First export.
	if res := Run(fs, []int{0}, Options{Dir: dir, BaseName: "cat"}, nil); res.Written != 1 {
		t.Fatalf("initial export: %+v", res)
	}
	before, _ := os.ReadFile(target)

	// Without overwrite: skipped, file untouched, not failed.
	fs2 := []gifdec.Frame{makeFrame(t, 2, 2, 200)}
	res := Run(fs2, []int{0}, Options{Dir: dir, BaseName: "cat"}, nil)
	if res.Written != 0 || len(res.Failed) != 0 {
		t.Fatalf("skip run: %+v", res)
	}
	after, _ := os.ReadFile(target)
	if string(before) != string(after) {
		t.Fatal("file was modified without overwrite")
	}

	// With overwrite: written and content replaced.
	res = Run(fs2, []int{0}, Options{Dir: dir, BaseName: "cat", Overwrite: true}, nil)
	if res.Written != 1 || len(res.Failed) != 0 {
		t.Fatalf("overwrite run: %+v", res)
	}
	after2, _ := os.ReadFile(target)
	if string(before) == string(after2) {
		t.Fatal("file was not overwritten")
	}
}

func TestExisting(t *testing.T) {
	dir := t.TempDir()
	fs := []gifdec.Frame{makeFrame(t, 2, 2, 1), makeFrame(t, 2, 2, 2)}
	Run(fs, []int{1}, Options{Dir: dir, BaseName: "cat"}, nil)

	got := Existing(Options{Dir: dir, BaseName: "cat"}, []int{0, 1}, 2)
	if len(got) != 1 || filepath.Base(got[0]) != "cat_frame_0002.png" {
		t.Fatalf("Existing = %v", got)
	}
}

func TestSanitise(t *testing.T) {
	cases := map[string]string{
		`cat`:                 "cat",
		`a<b>c:d"e/f\g|h?i*j`: "a_b_c_d_e_f_g_h_i_j",
		"":                    "frames",
		"a\tb\x01c":           "a_b_c",
	}
	for in, want := range cases {
		if got := Sanitise(in); got != want {
			t.Errorf("Sanitise(%q) = %q, want %q", in, got, want)
		}
	}
	// The sanitised base name is what lands on disk.
	dir := t.TempDir()
	fs := []gifdec.Frame{makeFrame(t, 2, 2, 1)}
	Run(fs, []int{0}, Options{Dir: dir, BaseName: `bad/name<here`}, nil)
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "bad_name_here_frame_0001.png" {
		t.Fatalf("dir = %v", entries)
	}
}

func TestRunUnwritableDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows: chmod does not make directory unwritable")
	}

	dir := t.TempDir()

	ro := filepath.Join(dir, "readonly")
	if err := os.Mkdir(ro, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(ro, 0o700)

	fs := []gifdec.Frame{makeFrame(t, 2, 2, 1)}
	res := Run(fs, []int{0}, Options{Dir: ro, BaseName: "cat"}, nil)
	if res.Written != 0 || len(res.Failed) != 1 {
		t.Fatalf("result = %+v", res)
	}
	// No .tmp files left behind.
	entries, err := os.ReadDir(ro)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

func TestRunOutOfRange(t *testing.T) {
	dir := t.TempDir()
	fs := []gifdec.Frame{makeFrame(t, 2, 2, 1)}
	res := Run(fs, []int{5}, Options{Dir: dir, BaseName: "cat"}, nil)
	if res.Written != 0 || len(res.Failed) != 1 {
		t.Fatalf("result = %+v", res)
	}
}
