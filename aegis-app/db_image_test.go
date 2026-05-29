package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// -----------------------------------------------------------------------------
// Test helpers
// -----------------------------------------------------------------------------

// newOpaqueImage returns an RGBA image filled with a single fully-opaque color.
func newOpaqueImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	red := color.RGBA{R: 200, G: 50, B: 50, A: 0xff}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, red)
		}
	}
	return img
}

// newTransparentImage returns an RGBA image with at least one fully-transparent pixel.
func newTransparentImage(w, h int) *image.RGBA {
	img := newOpaqueImage(w, h)
	img.Set(0, 0, color.RGBA{R: 0, G: 0, B: 0, A: 0})
	return img
}

// -----------------------------------------------------------------------------
// hasTransparency
// -----------------------------------------------------------------------------

func TestHasTransparencyOpaque(t *testing.T) {
	img := newOpaqueImage(8, 8)
	if hasTransparency(img) {
		t.Error("fully opaque image should not be reported as transparent")
	}
}

func TestHasTransparencyDetectsAlpha(t *testing.T) {
	img := newTransparentImage(8, 8)
	if !hasTransparency(img) {
		t.Error("image with alpha=0 pixel should be reported as transparent")
	}
}

func TestHasTransparencyDetectsPartialAlpha(t *testing.T) {
	img := newOpaqueImage(4, 4)
	// Single pixel with alpha=200 (still <0xffff after RGBA() expansion).
	img.Set(2, 2, color.RGBA{R: 100, G: 100, B: 100, A: 200})
	if !hasTransparency(img) {
		t.Error("image with partial alpha should be reported as transparent")
	}
}

// -----------------------------------------------------------------------------
// resizeImageIfNeeded
// -----------------------------------------------------------------------------

func TestResizeImageIfNeededSmallStaysUnchanged(t *testing.T) {
	src := newOpaqueImage(100, 50)
	out := resizeImageIfNeeded(src, 200)
	if out != image.Image(src) {
		// Implementation returns the same pointer when no resize is needed.
		t.Errorf("expected source image to be returned as-is, got bounds %v", out.Bounds())
	}
}

func TestResizeImageIfNeededRespectsAspectRatio(t *testing.T) {
	src := newOpaqueImage(2000, 1000)
	out := resizeImageIfNeeded(src, 500)
	bounds := out.Bounds()
	if bounds.Dx() != 500 {
		t.Errorf("expected width 500, got %d", bounds.Dx())
	}
	if bounds.Dy() != 250 {
		t.Errorf("expected height 250 (preserve 2:1 ratio), got %d", bounds.Dy())
	}
}

func TestResizeImageIfNeededTallImageScalesByHeight(t *testing.T) {
	src := newOpaqueImage(500, 2000)
	out := resizeImageIfNeeded(src, 400)
	bounds := out.Bounds()
	if bounds.Dy() != 400 {
		t.Errorf("expected height 400, got %d", bounds.Dy())
	}
	if bounds.Dx() != 100 {
		t.Errorf("expected width 100, got %d", bounds.Dx())
	}
}

func TestResizeImageIfNeededZeroOrNegativeMaxEdge(t *testing.T) {
	src := newOpaqueImage(100, 100)
	if out := resizeImageIfNeeded(src, 0); out != image.Image(src) {
		t.Error("maxEdge=0 should return source unchanged")
	}
	if out := resizeImageIfNeeded(src, -10); out != image.Image(src) {
		t.Error("negative maxEdge should return source unchanged")
	}
}

func TestResizeImageIfNeededAtLeastOnePixel(t *testing.T) {
	// Force computed dim < 1 by giving a very large source and tiny maxEdge.
	src := newOpaqueImage(10000, 1)
	out := resizeImageIfNeeded(src, 10)
	bounds := out.Bounds()
	if bounds.Dx() < 1 || bounds.Dy() < 1 {
		t.Errorf("expected min 1x1, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

// -----------------------------------------------------------------------------
// encodeImageForStorage
// -----------------------------------------------------------------------------

func TestEncodeImageForStoragePNGWithTransparency(t *testing.T) {
	src := newTransparentImage(8, 8)
	bytesOut, mime, err := encodeImageForStorage(src, "image/png")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if mime != "image/png" {
		t.Errorf("expected PNG to be preserved when transparency present, got %q", mime)
	}
	// Verify decodable as PNG.
	if _, err := png.Decode(bytes.NewReader(bytesOut)); err != nil {
		t.Errorf("output not decodable as PNG: %v", err)
	}
}

func TestEncodeImageForStoragePNGNoTransparencyDowngradesToJPEG(t *testing.T) {
	src := newOpaqueImage(16, 16)
	_, mime, err := encodeImageForStorage(src, "image/png")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if mime != "image/jpeg" {
		t.Errorf("expected opaque PNG to be downgraded to JPEG, got %q", mime)
	}
}

func TestEncodeImageForStorageDefaultsToJPEG(t *testing.T) {
	src := newOpaqueImage(8, 8)
	_, mime, err := encodeImageForStorage(src, "")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if mime != "image/jpeg" {
		t.Errorf("expected default JPEG, got %q", mime)
	}
}

func TestEncodeImageForStorageUnknownMIMEFallsBackToJPEG(t *testing.T) {
	src := newOpaqueImage(8, 8)
	_, mime, err := encodeImageForStorage(src, "image/webp")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if mime != "image/jpeg" {
		t.Errorf("unknown MIME should fall back to JPEG, got %q", mime)
	}
}

func TestEncodeImageForStorageMIMECaseInsensitive(t *testing.T) {
	src := newTransparentImage(8, 8)
	_, mime, err := encodeImageForStorage(src, "IMAGE/PNG")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if mime != "image/png" {
		t.Errorf("case-insensitive PNG match expected, got %q", mime)
	}
}

// -----------------------------------------------------------------------------
// Round-trip prepareImageAssets with a real PNG (covers more of the path)
// -----------------------------------------------------------------------------

func TestPrepareImageAssetsRealPNG(t *testing.T) {
	// Encode a synthetic 4x4 transparent PNG.
	src := newTransparentImage(4, 4)
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatalf("encode test PNG: %v", err)
	}

	mainBytes, mainMime, w, h, thumbBytes, thumbMime, tw, th, err := prepareImageAssets(buf.Bytes(), "image/png")
	if err != nil {
		t.Fatalf("prepareImageAssets: %v", err)
	}
	if w != 4 || h != 4 {
		t.Errorf("dimensions mismatch: %dx%d", w, h)
	}
	if len(mainBytes) == 0 {
		t.Error("main bytes should be populated")
	}
	if mainMime != "image/png" {
		t.Errorf("transparent PNG should keep PNG mime, got %q", mainMime)
	}
	// The 4x4 source is below thumbnail threshold (320), so thumb dims match source.
	if tw != 4 || th != 4 {
		t.Errorf("thumb dims for tiny source should equal source, got %dx%d", tw, th)
	}
	if thumbMime != "image/jpeg" {
		t.Errorf("thumb is encoded as JPEG by prepareImageAssets, got %q", thumbMime)
	}
	if len(thumbBytes) == 0 {
		t.Error("thumb bytes should be populated")
	}
}
