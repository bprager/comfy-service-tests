package imaging

func trimText(text string, max int) string {
	if len(text) <= max {
		return text
	}
	if max < 3 {
		return text[:max]
	}
	return text[:max-3] + "..."
}
