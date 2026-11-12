package shape

import (
	"strings"
	"testing"

	"github.com/KogenAI/kogen-verification/verify"
)

// Cases fl-l01 and kd-l02 of the iteration-0 component eval set. The model listed the ids of
// "Studio issue <id>." and "Blueprint <slug> r2 owns the architecture." as required literals and
// Shape refused a package that paraphrases them. The commit id of a request line that shows real
// output ("Commit: 1eb6a4ca ...") is a product literal and must stay required.
const provenanceRequest = "Studio issue 932bc96f-4221-4234-aab1-6a910b37f31d.\n\nBlueprint studio-product-redesign-20261109 r2 owns the architecture.\n\nAdd an additive, rebuildable current projection.\n\nCommit: 1eb6a4ca Add Kogen configuration\n\nShow `Blueprint studio-product-redesign-20260101 r1` in the footer.\n\nStudio issue 11111111-2222-3333-4444-555555555555.\n\nAlso print 11111111-2222-3333-4444-555555555555 in the footer."

func TestProvenanceIdsAreNotProductLiterals(t *testing.T) {
	parallel(t)
	files := []packageFile{{"intent.md", "This delivers the originating issue; the product-redesign blueprint owns the architecture."}}
	sources := map[string]string{"request": provenanceRequest}
	red := verify.NewRedactor(nil, "")
	cases := []struct {
		name, literal, quote, status string
	}{
		{"fl-l01 issue id", "932bc96f-4221-4234-aab1-6a910b37f31d", "Studio issue 932bc96f-4221-4234-aab1-6a910b37f31d.", "informal"},
		{"fl-l01 blueprint id", "studio-product-redesign-20261109", "Blueprint studio-product-redesign-20261109 r2 owns the architecture.", "informal"},
		{"fl-l01 whole provenance line", "Blueprint studio-product-redesign-20261109 r2 owns the architecture", "Blueprint studio-product-redesign-20261109 r2 owns the architecture.", "informal"},
		{"kd-l02 commit line", "Commit: 1eb6a4ca Add Kogen configuration", "Commit: 1eb6a4ca Add Kogen configuration", "missing"},
		{"kd-l02 commit id", "1eb6a4ca", "Commit: 1eb6a4ca Add Kogen configuration", "missing"},
		{"id the request also asks to print", "11111111-2222-3333-4444-555555555555", "Also print 11111111-2222-3333-4444-555555555555 in the footer.", "missing"},
		{"id shown in backticks", "studio-product-redesign-20260101", "Show `Blueprint studio-product-redesign-20260101 r1` in the footer.", "missing"},
	}
	for _, c := range cases {
		it := litItem(c.literal, "request", c.quote, "required", "")
		event, lines := literalOutcome([]literalItem{it}, sources, files, red)
		got := event["items"].([]any)[0].(map[string]any)["status"]
		if got != c.status {
			t.Errorf("%s: status %v, want %s", c.name, got, c.status)
		}
		if refused := len(lines) > 0; refused != (c.status == "missing") {
			t.Errorf("%s: refused %v, lines %q", c.name, refused, lines)
		}
	}
}

func TestProvenanceLineNeedsEveryOccurrence(t *testing.T) {
	parallel(t)
	for request, want := range map[string]bool{
		"Studio issue 932bc96f-4221.\n\nBuild it.":                       true,
		"Blueprint studio-redesign-20261109 r2 owns the architecture.":   true,
		"Issue 932bc96f-4221.\n\nStudio issue 932bc96f-4221 again.":      true,
		"Studio issue 932bc96f-4221.\n\nPrint 932bc96f-4221.":            false,
		"Studio issue 932bc96f-4221.\nPrint 932bc96f-4221 as the title.": false,
		"Build it.": false,
	} {
		lit := "932bc96f-4221"
		if strings.Contains(request, "studio-redesign") {
			lit = "studio-redesign-20261109"
		}
		if got := provenanceOnly(request, lit); got != want {
			t.Errorf("%q: %v, want %v", request, got, want)
		}
	}
	if provenanceOnly("Studio issue abc.", "abc") {
		t.Error("a literal without a digit is not an id")
	}
}
