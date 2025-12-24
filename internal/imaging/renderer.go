package imaging

type RenderOptions struct {
	Width      int
	Height     int
	Prompt     string
	Negative   string
	Checkpoint string
	Seed       int64
}

func RenderPlaceholder(opts RenderOptions) ([]byte, error) {
	return renderPlaceholder(opts)
}
