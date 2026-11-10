package shape

import (
	"strings"
	"testing"
)

func TestControlsFailureMessage(t *testing.T) {
	tests := []struct{ input, want string }{
		{"\n \t\n", ""},
		{" expected  true \n", "expected true"},
		{"\n command: \n note  one \n x/file.go:12:3: bad\n note two \n y.go:4: bad", "command: | x/file.go:12:3: bad | y.go:4: bad | note one"},
		{strings.Repeat("界", 401), strings.Repeat("界", 400)},
	}
	for _, tt := range tests {
		if got := controlsFailureMessage(tt.input); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
	for _, width := range []int{1, 127, 128, 129, 130} {
		input := "first" + strings.Repeat("\n"+strings.Repeat("界", width), 50)
		expected := "first"
		for i := 0; i < 3; i++ {
			n := 400 - len([]rune(expected)) - 3
			if n <= 0 {
				break
			}
			if n > width {
				n = width
			}
			expected += " | " + strings.Repeat("界", n)
		}
		if got := controlsFailureMessage(input); got != expected {
			t.Errorf("width %d: got %q, want %q", width, got, expected)
		}
	}
	// No separator fits when the first line consumes the remaining budget.
	input := strings.Repeat("a", 397) + "\nnext"
	if got := controlsFailureMessage(input); got != strings.Repeat("a", 397) {
		t.Errorf("empty separator: %q", got)
	}
}
