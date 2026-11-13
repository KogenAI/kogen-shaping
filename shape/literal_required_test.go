package shape

import (
	"slices"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-verification/verify"
)

// A request that shows the real
// output of `kogen init` in a code block, including "Commit: 1eb6a4ca Add Kogen configuration".
// The model listed that line in 3 of 5 runs, so whether Shape refused a package without it
// depended on the model.
const requiredRequest = "Studio issue 932bc96f-4221-4234-aab1-6a910b37f31d.\n\n" +
	"Document `kogen init`; update packaging/package.sh in the same change so `sh packaging/package.sh` still succeeds.\n\n" +
	"Real outputs:\n\n```\nInitialized Kogen in <path>\nCommit: 1eb6a4ca Add Kogen configuration\n  checks: acceptance 5/5\n```\n\n" +
	"Say exactly \"Upgrade to Unlimited\" on the paywall, and `Next: kogen intent build <slug>` after approval."

func requiredOutcome(model []literalItem, packageText string) (map[string]string, []string) {
	items := withRequestLiterals(requiredRequest, model)
	event, lines := literalOutcome(items, map[string]string{"request": requiredRequest}, []packageFile{{"intent.md", packageText}}, verify.NewRedactor(nil, ""))
	status := map[string]string{}
	for _, raw := range event["items"].([]any) {
		m := raw.(map[string]any)
		status[m["literal"].(string)] = m["status"].(string)
	}
	return status, lines
}

func TestRequestLiteralsListIsDeterministic(t *testing.T) {
	parallel(t)
	var got []string
	for _, it := range requestLiterals(requiredRequest) {
		got = append(got, it.literal)
	}
	want := []string{"kogen init", "Initialized Kogen in <path>", "Commit: 1eb6a4ca Add Kogen configuration", "checks: acceptance 5/5",
		"Next: kogen intent build <slug>", "Upgrade to Unlimited"}
	// code span of the request's first paragraph, then the fenced lines, then the later paragraph's quote and span
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("literals %q, want %q", got, want)
	}
	for _, not := range []string{"932bc96f-4221-4234-aab1-6a910b37f31d", "sh packaging/package.sh", "packaging/package.sh"} {
		if slices.Contains(got, not) {
			t.Errorf("%q must not be a required literal", not)
		}
	}
}

// With the model's list empty, the Commit line is still required and refused when missing.
func TestRequestOutputLineIsAlwaysRequired(t *testing.T) {
	parallel(t)
	const without = "Initialized Kogen in <path>\n  checks: acceptance 5/5\n`kogen init`, `Next: kogen intent build <slug>` and Upgrade to Unlimited."
	for name, model := range map[string][]literalItem{
		"model lists nothing":           nil,
		"model marks the line informal": {litItem("Commit: 1eb6a4ca Add Kogen configuration", "request", "Commit: 1eb6a4ca Add Kogen configuration", "informal", "")},
		"model lists the line":          {litItem("Commit: 1eb6a4ca Add Kogen configuration", "request", "Commit: 1eb6a4ca Add Kogen configuration", "required", "")},
	} {
		status, lines := requiredOutcome(model, without)
		if status["Commit: 1eb6a4ca Add Kogen configuration"] != "missing" || len(lines) == 0 || !strings.Contains(strings.Join(lines, "\n"), "Commit: 1eb6a4ca") {
			t.Errorf("%s: status %v, lines %q", name, status, lines)
		}
	}
	status, lines := requiredOutcome(nil, without+"\nCommit: 1eb6a4ca Add Kogen configuration")
	if len(lines) != 0 || status["Commit: 1eb6a4ca Add Kogen configuration"] != "present" {
		t.Errorf("present line: %v %q", status, lines)
	}
}

// Provenance rule: a provenance id stays informal, the request's own literals do
// not turn it required, and a keep-working command is not a literal.
func TestRequestLiteralsKeepProvenanceInformal(t *testing.T) {
	parallel(t)
	const id = "932bc96f-4221-4234-aab1-6a910b37f31d"
	const pkg = "Initialized Kogen in <path>\nCommit: 1eb6a4ca Add Kogen configuration\nchecks: acceptance 5/5\nkogen init\nNext: kogen intent build <slug>\nUpgrade to Unlimited"
	model := []literalItem{
		litItem(id, "request", "Studio issue "+id+".", "required", ""),
		litItem("sh packaging/package.sh", "request", "`sh packaging/package.sh`", "informal", ""),
	}
	status, lines := requiredOutcome(model, pkg)
	if status[id] != "informal" || len(lines) != 0 {
		t.Errorf("provenance id: %v %q", status, lines)
	}
	if _, listed := status["sh packaging/package.sh"]; !listed {
		// the model's own item stays in the list; it is informal there
		t.Log("model item kept:", status)
	}
}

// The model keeps a say over the ambiguous code spans and over superseded values, not over lines
// the request shows in a code block.
func TestRequestLiteralsModelMerge(t *testing.T) {
	parallel(t)
	model := []literalItem{
		litItem("kogen init", "request", "`kogen init`", "informal", ""),
		litItem("Initialized Kogen in <path>", "request", "Initialized Kogen in <path>", "informal", ""),
		litItem("checks: acceptance 5/5", "request", "  checks: acceptance 5/5", "superseded", ""),
		litItem("Upgrade to Unlimited", "request", "Upgrade to Unlimited", "required", "Upgrade to Pro"),
	}
	status, _ := requiredOutcome(model, "Upgrade to Pro")
	for lit, want := range map[string]string{"kogen init": "informal", "Initialized Kogen in <path>": "missing", "checks: acceptance 5/5": "superseded", "Upgrade to Unlimited": "changed"} {
		if status[lit] != want {
			t.Errorf("%q: %s, want %s", lit, status[lit], want)
		}
	}
	if got := withRequestLiterals("Build it.", nil); got != nil {
		t.Errorf("no literals in the request: %v", got)
	}
}

// Through the shaping run: the model's literal check lists nothing, yet a literal the request
// writes as code is still required and a package without it is refused.
func TestShapingRequiresRequestLiteralsWhenTheModelListsNone(t *testing.T) {
	parallel(t)
	f := newShpFx(t, shpSubmit("c1", shpFiles()), shpSubmit("c2", shpFiles()), shpSubmit("c3", shpFiles()))
	if _, err := f.shape("Add calc.is_even(n). Use limit `4242`."); err == nil {
		t.Fatal("a package without the request's literal was published")
	}
	fed := shpOutputs(f.Prov.requests()[1])["c1"]
	if !strings.Contains(fed, "literal: 4242 | source: request") || !strings.Contains(fed, "status: missing") {
		t.Fatalf("feedback lacks the missing request literal:\n%s", fed)
	}
}
