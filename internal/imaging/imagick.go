//go:build imagick

package imaging

import (
	"bytes"
	"fmt"

	"github.com/gographics/imagick/imagick"
)

func renderPlaceholder(opts RenderOptions) ([]byte, error) {
	width := uint(opts.Width)
	height := uint(opts.Height)
	if width == 0 {
		width = 512
	}
	if height == 0 {
		height = 512
	}

	imagick.Initialize()
	defer imagick.Terminate()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	bg := imagick.NewPixelWand()
	bg.SetColor("#0f151b")
	if err := mw.NewImage(width, height, bg); err != nil {
		return nil, err
	}

	if err := mw.SetImageFormat("png"); err != nil {
		return nil, err
	}

	draw := imagick.NewDrawingWand()
	defer draw.Destroy()

	accent := imagick.NewPixelWand()
	accent.SetColor("#ffb454")
	draw.SetFillColor(accent)
	draw.SetFontSize(18)
	draw.SetFontWeight(600)

	label := fmt.Sprintf("checkpoint: %s", opts.Checkpoint)
	_ = mw.AnnotateImage(draw, 24, 48, 0, label)
	_ = mw.AnnotateImage(draw, 24, 76, 0, "prompt: "+trimText(opts.Prompt, 60))

	blob := mw.GetImageBlob()
	return bytes.Clone(blob), nil
}
