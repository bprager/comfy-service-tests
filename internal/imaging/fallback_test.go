package imaging

import (
	"bytes"
	"image/png"
	"testing"
)

func TestRenderPlaceholderPNG(t *testing.T) {
	payload, err := RenderPlaceholder(RenderOptions{Width: 320, Height: 240, Prompt: "hello"})
	if err != nil {
		t.Fatalf("render placeholder: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 320 || bounds.Dy() != 240 {
		t.Fatalf("unexpected size: %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestRenderPlaceholderDefaults(t *testing.T) {
	payload, err := RenderPlaceholder(RenderOptions{})
	if err != nil {
		t.Fatalf("render placeholder: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 512 || bounds.Dy() != 512 {
		t.Fatalf("unexpected default size: %dx%d", bounds.Dx(), bounds.Dy())
	}
}
