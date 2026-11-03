package shape

import (
	"slices"
	"testing"
)

func TestAssumptionLines(t *testing.T) {
	parallel(t)
	intent := "## Decisions\n- A. (source: request)\n- B. (assumption: x) (source: request)\n\n## Tests\n- C (assumption: y)\n"
	if got := assumptionLines(intent); !slices.Equal(got, []string{"B. (assumption: x)"}) {
		t.Errorf("got %q", got)
	}
	if got := assumptionLines("## Decisions\n- A. (source: request)\n"); got != nil {
		t.Errorf("none: %q", got)
	}
}

func TestContainsNormalized(t *testing.T) {
	parallel(t)
	cases := []struct {
		hay, needle string
		want        bool
	}{
		{"Use only the\nGo standard library.", "Use only the Go standard library.", true},
		{"Use only the\r\nGo  standard\tlibrary.", "Use only the Go standard library.", true},
		{"Use only the Go standard library.", "Use only the Go  library.", false},
		{"Use only the Go standard library.", "use only the go standard library.", false},
	}
	for _, c := range cases {
		if got := containsNormalized(c.hay, c.needle); got != c.want {
			t.Errorf("containsNormalized(%q,%q)=%v", c.hay, c.needle, got)
		}
	}
}
