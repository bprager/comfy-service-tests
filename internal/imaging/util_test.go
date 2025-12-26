package imaging

import "testing"

func TestTrimText(t *testing.T) {
	if got := trimText("short", 10); got != "short" {
		t.Fatalf("unexpected text: %s", got)
	}
	if got := trimText("truncate-me", 4); got != "t..." {
		t.Fatalf("unexpected truncation: %s", got)
	}
	if got := trimText("xyz", 2); got != "xy" {
		t.Fatalf("unexpected short truncation: %s", got)
	}
}
