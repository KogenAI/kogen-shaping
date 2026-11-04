package consumes

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/KogenAI/kogen-verification/verify"
)

// Signatures kogen-shaping's calls rely on.
var (
	_ func(string) string = verify.Cut
	_ func(string) string = verify.Diagnostic
)

// verify.Cut bounds the text kogen-shaping hands the model or keeps in a record to 16 KiB:
// the first 4 KiB and the last part with the omitted byte count between, whole UTF-8 only.
func TestCut(t *testing.T) {
	short := strings.Repeat("a", 16384)
	if got := verify.Cut(short); got != short {
		t.Errorf("verify.Cut: a text of exactly 16384 bytes must come back unchanged, got %d bytes", len(got))
	}
	text := strings.Repeat("0123456789", 3000) // 30000 bytes
	got := verify.Cut(text)
	if len(got) != 16384 {
		t.Errorf("verify.Cut: a long text must be cut to 16384 bytes, got %d", len(got))
	}
	head, rest, ok := strings.Cut(got, "\n[... ")
	if !ok || head != text[:4096] {
		t.Fatalf("verify.Cut: the first 4096 bytes must stay, then the omitted marker; got head of %d bytes (marker found: %v)", len(head), ok)
	}
	var omitted int
	if _, err := fmt.Sscanf(rest, "%d bytes omitted ...]\n", &omitted); err != nil {
		t.Fatalf("verify.Cut: the marker after %q is not \"[... N bytes omitted ...]\": %v", head[len(head)-4:], err)
	}
	_, tail, _ := strings.Cut(rest, "bytes omitted ...]\n")
	if !strings.HasSuffix(text, tail) || omitted != len(text)-len(head)-len(tail) {
		t.Errorf("verify.Cut: the tail must be the end of the text and the marker must count the %d omitted bytes, got %d", len(text)-len(head)-len(tail), omitted)
	}

	multi := strings.Repeat("é€", 9000) // 2- and 3-byte runes, never cut inside one
	if got := verify.Cut(multi); !utf8.ValidString(got) || !strings.Contains(got, "bytes omitted") {
		t.Errorf("verify.Cut: a cut of multi-byte text must stay valid UTF-8 and carry the marker")
	}
}

// verify.Diagnostic is the excerpt a refusal shows: whole up to 60 lines, else the first 30
// and the last 20, then the bound of verify.Cut.
func TestDiagnostic(t *testing.T) {
	var lines []string
	for i := 1; i <= 60; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	whole := strings.Join(lines, "\n")
	if got := verify.Diagnostic(whole + "\n\n  \n"); got != whole {
		t.Errorf("verify.Diagnostic: 60 lines must stay whole with the trailing blank space trimmed, got %q", got)
	}

	for i := 61; i <= 100; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	want := strings.Join(lines[:30], "\n") + "\n[... 50 lines omitted ...]\n" + strings.Join(lines[80:], "\n")
	if got := verify.Diagnostic(strings.Join(lines, "\n")); got != want {
		t.Errorf("verify.Diagnostic: 100 lines must give the first 30, the omitted count (50) and the last 20, got:\n%s", got)
	}

	huge := verify.Diagnostic(strings.Repeat("x", 50000))
	if len(huge) != 16384 || !strings.Contains(huge, "bytes omitted") {
		t.Errorf("verify.Diagnostic: one enormous line must be bounded by verify.Cut to 16384 bytes, got %d", len(huge))
	}
}
