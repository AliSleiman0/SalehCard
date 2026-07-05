package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
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
