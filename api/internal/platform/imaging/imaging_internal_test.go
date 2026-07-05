package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

// fixturePNG encodes a small w×h RGBA image with a semi-transparent pixel, so
// callers can assert alpha gets composited onto white.
func fixturePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 128})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode fixture png: %v", err)
	}
	return buf.Bytes()
}

// fixtureJPEG encodes an opaque w×h image, exercising the no-alpha (draw.Src)
// branch of scaleOnto that fixturePNG (which always carries alpha) never hits.
func fixtureJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 120, B: 40, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode fixture jpeg: %v", err)
	}
	return buf.Bytes()
}

func TestProcess_PNGWithAlpha_ProducesJPEGs(t *testing.T) {
	data := fixturePNG(t, 300, 200)

	display, thumb, err := Process(data)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	displayCfg, format, err := image.DecodeConfig(bytes.NewReader(display))
	if err != nil {
		t.Fatalf("decode display: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("display format = %q, want jpeg", format)
	}
	if displayCfg.Width != 300 || displayCfg.Height != 200 {
		t.Fatalf("display dims = %dx%d, want 300x200 (no upscale expected)", displayCfg.Width, displayCfg.Height)
	}

	thumbCfg, format, err := image.DecodeConfig(bytes.NewReader(thumb))
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("thumb format = %q, want jpeg", format)
	}
	if thumbCfg.Width != thumbMaxEdge {
		t.Fatalf("thumb width = %d, want %d", thumbCfg.Width, thumbMaxEdge)
	}
}

func TestProcess_OversizedDimensions_Rejected(t *testing.T) {
	orig := maxDecodeDim
	maxDecodeDim = 10
	defer func() { maxDecodeDim = orig }()

	data := fixturePNG(t, 20, 20)

	_, _, err := Process(data)
	if err != ErrTooLarge {
		t.Fatalf("Process error = %v, want ErrTooLarge", err)
	}
}

func TestProcess_NonImageBytes_Rejected(t *testing.T) {
	_, _, err := Process([]byte("this is not an image, just some plain text bytes"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("Process error = %v, want ErrUnsupportedFormat", err)
	}
}

func TestProcess_OpaqueJPEG_ProducesJPEGs(t *testing.T) {
	// A small opaque JPEG: both outputs must decode as JPEG and, being under
	// 1024px, keep their source dimensions on the display image.
	data := fixtureJPEG(t, 320, 240)

	display, thumb, err := Process(data)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	displayCfg, format, err := image.DecodeConfig(bytes.NewReader(display))
	if err != nil {
		t.Fatalf("decode display: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("display format = %q, want jpeg", format)
	}
	if displayCfg.Width != 320 || displayCfg.Height != 240 {
		t.Fatalf("display dims = %dx%d, want 320x240 (no upscale)", displayCfg.Width, displayCfg.Height)
	}

	if _, format, err = image.DecodeConfig(bytes.NewReader(thumb)); err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("thumb format = %q, want jpeg", format)
	}
}

func TestProcess_Downscale_LandscapeToDisplayEdge(t *testing.T) {
	// 2000×1500 landscape: the longer (width) edge governs → display width 1024,
	// height scaled proportionally; thumb longer edge 256.
	data := fixturePNG(t, 2000, 1500)

	display, thumb, err := Process(data)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	dcfg, _, err := image.DecodeConfig(bytes.NewReader(display))
	if err != nil {
		t.Fatalf("decode display: %v", err)
	}
	if dcfg.Width != displayMaxEdge {
		t.Fatalf("display width = %d, want %d", dcfg.Width, displayMaxEdge)
	}
	if dcfg.Height != 768 { // 1500 * 1024 / 2000
		t.Fatalf("display height = %d, want 768 (aspect preserved)", dcfg.Height)
	}

	tcfg, _, err := image.DecodeConfig(bytes.NewReader(thumb))
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	if tcfg.Width != thumbMaxEdge {
		t.Fatalf("thumb width = %d, want %d", tcfg.Width, thumbMaxEdge)
	}
}

func TestProcess_Downscale_PortraitToDisplayEdge(t *testing.T) {
	// 400×1200 portrait: the longer (height) edge governs → display height 1024,
	// width scaled proportionally and clamped ≥ 1.
	data := fixturePNG(t, 400, 1200)

	display, _, err := Process(data)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	dcfg, _, err := image.DecodeConfig(bytes.NewReader(display))
	if err != nil {
		t.Fatalf("decode display: %v", err)
	}
	if dcfg.Height != displayMaxEdge {
		t.Fatalf("display height = %d, want %d (longer edge governs)", dcfg.Height, displayMaxEdge)
	}
	if dcfg.Width != 341 { // 400 * 1024 / 1200
		t.Fatalf("display width = %d, want 341 (aspect preserved)", dcfg.Width)
	}
	if dcfg.Width < 1 {
		t.Fatalf("display width = %d, want ≥ 1", dcfg.Width)
	}
}

func TestProcess_CorruptButSniffedAsImage_DecodeError(t *testing.T) {
	// A valid JPEG SOI marker so http.DetectContentType reports image/jpeg, then
	// garbage so the JPEG decode fails — proving a sniffed-but-undecodable file
	// yields a wrapped decode error, not ErrUnsupportedFormat / ErrTooLarge.
	data := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, []byte("not a real jpeg payload")...)

	_, _, err := Process(data)
	if err == nil {
		t.Fatal("Process: want a decode error, got nil")
	}
	if err == ErrUnsupportedFormat || err == ErrTooLarge {
		t.Fatalf("Process error = %v, want a wrapped decode error", err)
	}
	if !strings.Contains(err.Error(), "imaging: decode") {
		t.Fatalf("Process error = %q, want it to wrap an \"imaging: decode\" failure", err)
	}
}
