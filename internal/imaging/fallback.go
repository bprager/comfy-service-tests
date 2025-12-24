//go:build !imagick

package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

func renderPlaceholder(opts RenderOptions) ([]byte, error) {
	width := opts.Width
	height := opts.Height
	if width <= 0 {
		width = 512
	}
	if height <= 0 {
		height = 512
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8((x + y) % 255)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	overlay := image.NewUniform(color.RGBA{R: 12, G: 14, B: 18, A: 200})
	draw.Draw(img, image.Rect(20, 20, width-20, 120), overlay, image.Point{}, draw.Over)

	label := fmt.Sprintf("checkpoint: %s", opts.Checkpoint)
	drawSimpleText(img, 32, 50, label)
	drawSimpleText(img, 32, 80, "prompt: "+trimText(opts.Prompt, 48))

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawSimpleText(img *image.RGBA, x, y int, text string) {
	for i, ch := range []byte(text) {
		px := x + i*6
		for dy := 0; dy < 5; dy++ {
			for dx := 0; dx < 3; dx++ {
				if (ch+byte(dx)+byte(dy))%7 == 0 {
					img.SetRGBA(px+dx, y+dy, color.RGBA{R: 240, G: 196, B: 120, A: 255})
				}
			}
		}
	}
}
