package shape

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequestFindingsOutcomes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "calc.js"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	request := func(items ...string) string {
		return "Add it to `calc.js`.\n\n## Acceptance\n" + strings.Join(items, "\n") + "\n"
	}
	cases := []struct {
		name, request string
		want          []string // substrings, one per finding line
	}{
		{"clean", request("1. A. Outcome: is_even(4) returns true.", "2. B. Outcome: is_even(3) returns false."), nil},
		{"same words", request("1. A. Outcome: is_even(4) returns true.", "2. B. Outcome: is_even(4) returns true."), []string{"scenarios 1 and 2"}},
		{"other words", request("1. A. Outcome: is_even(4) returns true.", "2. B. Outcome: true is returned by is_even(4)."), []string{"scenarios 1 and 2"}},
		{"no outcome", request("1. A. Outcome: x is 1.", "2. B."), []string{"scenario 2 "}},
		{"empty outcome", request("1. A. Outcome:"), []string{"scenario 1 "}},
		{"missing path", "Use `src/none.py` and `calc.js`.\n", []string{"`src/none.py`"}},
		{"routes and code are not paths", "Serve `/api/users`, call `calc.add(1)` and `1.5`, see `calc.js`.\n", nil},
		{"outside the acceptance section", "## Notes\n1. one\n2. two\n", nil},
	}
	for _, c := range cases {
		got := requestFindings(root, c.request)
		if len(got) != len(c.want) {
			t.Errorf("%s: findings %q, want %d", c.name, got, len(c.want))
			continue
		}
		for i, w := range c.want {
			if !strings.Contains(got[i], w) {
				t.Errorf("%s: finding %q lacks %q", c.name, got[i], w)
			}
		}
	}
}

func TestRepairProtectedDemands(t *testing.T) {
	protected := []string{"guard.txt"}
	got, files := repairProtectedDemands("Add x to `calc.js`. Then edit `guard.txt` to record it.\nSee `guard.txt`.\nDo not edit `guard.txt`.\n", protected)
	want := "Add x to `calc.js`.\n`guard.txt` is protected: the change must not touch it.\nSee `guard.txt`.\nDo not edit `guard.txt`.\n"
	if got != want || len(files) != 1 {
		t.Errorf("repaired %q (%v), want %q", got, files, want)
	}
	if got, files := repairProtectedDemands("Edit `other.txt`.\n", protected); got != "Edit `other.txt`.\n" || files != nil {
		t.Errorf("unprotected file changed: %q %v", got, files)
	}
}
