// Package imaging decodes an uploaded product image and re-encodes it into the
// two JPEG sizes the catalog stores: a 1024px "display" image and a 256px
// "thumbnail". It is pure Go stdlib plus golang.org/x/image (no cgo), matching
// the repo's lean-deps posture.
package imaging

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

// maxDecodeDim rejects images whose declared width or height (read from the
// header, before a full decode) exceeds this — a decode-bomb guard. A var, not
// a const, so tests can shrink it against small real fixtures.
var maxDecodeDim = 12000

const (
	displayMaxEdge = 1024
	thumbMaxEdge   = 256
	jpegQuality    = 80
)

// ErrUnsupportedFormat is returned when the sniffed content type is not one of
// image/jpeg, image/png, or image/webp.
var ErrUnsupportedFormat = errors.New("imaging: unsupported image format")

// ErrTooLarge is returned when the image's declared dimensions exceed
// [maxDecodeDim] on either edge.
var ErrTooLarge = errors.New("imaging: image dimensions too large")

// Process decodes data (sniffed as jpeg/png/webp — rejecting by content, not
// filename), composites any transparency onto white, and re-encodes it into two
// JPEGs at quality 80: display (max edge 1024px) and thumb (max edge 256px).
// Neither output is ever upscaled beyond the source's own dimensions.
func Process(data []byte) (display []byte, thumb []byte, err error) {
	contentType := http.DetectContentType(data)

	var cfgW, cfgH int
	switch contentType {
	case "image/jpeg":
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, nil, fmt.Errorf("imaging: decode config: %w", err)
		}
		cfgW, cfgH = cfg.Width, cfg.Height
	case "image/png":
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, nil, fmt.Errorf("imaging: decode config: %w", err)
		}
		cfgW, cfgH = cfg.Width, cfg.Height
	case "image/webp":
		cfg, err := webp.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, nil, fmt.Errorf("imaging: decode config: %w", err)
		}
		cfgW, cfgH = cfg.Width, cfg.Height
	default:
		return nil, nil, ErrUnsupportedFormat
	}
	if cfgW > maxDecodeDim || cfgH > maxDecodeDim {
		return nil, nil, ErrTooLarge
	}

	var img image.Image
	switch contentType {
	case "image/jpeg":
		img, err = jpeg.Decode(bytes.NewReader(data))
	case "image/png":
		img, err = png.Decode(bytes.NewReader(data))
	case "image/webp":
		img, err = webp.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return nil, nil, fmt.Errorf("imaging: decode: %w", err)
	}

	opaque := compositeOnWhite(img)

	display, err = resizeAndEncode(opaque, displayMaxEdge)
	if err != nil {
		return nil, nil, err
	}
	thumb, err = resizeAndEncode(opaque, thumbMaxEdge)
	if err != nil {
		return nil, nil, err
	}
	return display, thumb, nil
}

// compositeOnWhite flattens img onto an opaque white canvas of the same size,
// killing any alpha channel before JPEG encoding.
func compositeOnWhite(img image.Image) *image.RGBA {
	b := img.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, b, img, b.Min, draw.Over)
	return dst
}

// resizeAndEncode scales img down (never up) so its longer edge is at most
// maxEdge, using CatmullRom resampling, and JPEG-encodes the result at
// [jpegQuality].
func resizeAndEncode(img image.Image, maxEdge int) ([]byte, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	dstW, dstH := w, h
	if w > maxEdge || h > maxEdge {
		if w >= h {
			dstW = maxEdge
			dstH = int(float64(h) * float64(maxEdge) / float64(w))
		} else {
			dstH = maxEdge
			dstW = int(float64(w) * float64(maxEdge) / float64(h))
		}
		if dstW < 1 {
			dstW = 1
		}
		if dstH < 1 {
			dstH = 1
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Src, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("imaging: encode: %w", err)
	}
	return buf.Bytes(), nil
}
