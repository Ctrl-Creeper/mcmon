package app

import (
	"strings"
	"testing"
)

func TestStyleDefinesPolishedMotionSystem(t *testing.T) {
	b, err := staticFS.ReadFile("static/style.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(b)
	for _, want := range []string{
		"--ease-spring:",
		"--motion-fast:",
		"@keyframes page-in",
		"@keyframes card-in",
		"prefers-reduced-motion: reduce",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("style.css missing %q", want)
		}
	}
}
