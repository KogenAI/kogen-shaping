package shape

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const propIntent = "# T\n\n## Tests\n- needs human evaluation here\n\n## Decisions\n- The greeting feels welcoming  is judged  by a person, so it needs  Human Evaluation. (source: answer 1)\n- other\n\n## Feasibility\n- greeting is fine needs human evaluation\n"

func propEvidence() contractEvidence {
	return contractEvidence{
		scenarios:   []string{"Old", "New", "Other"},
		featurePath: "f.feature",
		expectations: map[string]string{
			"f.feature": "Then tests count 1 here",
			"s.js":      "assert(count >= 1) // tests exist",
		},
		intent:   propIntent,
		failures: map[string][]string{"New": {"Error: Cannot find module '../../slug'\n at x"}, "Other": {"AssertionError"}},
	}
}

func propReport(over func(p *propertyReport)) propertyReport {
	p := propertyReport{property: "p", request: "make it fast", kind: "new", observable: true, scenarios: []string{"New"}}
	if over != nil {
		over(&p)
	}
	return p
}

func TestPropertyProblems(t *testing.T) {
	parallel(t)
	const req = "Please make it  fast."
	cases := []struct {
		name string
		p    propertyReport
		want []string // reason:scenario
	}{
		{"covered and red", propReport(nil), nil},
		{"uncovered", propReport(func(p *propertyReport) { p.scenarios = []string{"Missing"} }), []string{"uncovered:"}},
		{"kept and passing is fine", propReport(func(p *propertyReport) { p.kind = "kept"; p.scenarios = []string{"Old", "Old"} }), nil},
		{"new passes unchanged", propReport(func(p *propertyReport) { p.scenarios = []string{"Old"} }), []string{"passes_unchanged:"}},
		{"missing", propReport(func(p *propertyReport) {
			p.issues = []propertyIssue{{propertyMissing, " New ", "Cannot find  module '../../slug'", "w"}, {propertyMissing, "New", "", "w"}, {propertyMissing, "Old", "x", "w"}, {propertyMissing, "New", "not there", "w"}}
		}), []string{"missing:New"}},
		{"missing needs kind new", propReport(func(p *propertyReport) {
			p.kind = "kept"
			p.issues = []propertyIssue{{propertyMissing, "New", "Cannot find module", "w"}}
		}), nil},
		{"test_count", propReport(func(p *propertyReport) {
			p.issues = []propertyIssue{{propertyTestCount, "New", "assert(count >= 1)", "w"}, {propertyTestCount, "New", "tests count 1", "w"}, {propertyTestCount, "Old", "assert(count", "w"}}
		}), []string{"test_count:New"}},
		{"claimed and recorded", propReport(func(p *propertyReport) {
			p.observable = false
			p.scenarios = []string{"Old", "Nope", "Old"}
			p.decision = "The greeting feels welcoming is judged by a person"
		}), []string{"claimed:Old"}},
		{"unrecorded blank", propReport(func(p *propertyReport) { p.observable = false; p.scenarios = nil }), []string{"unrecorded:"}},
		{"unrecorded outside Decisions", propReport(func(p *propertyReport) { p.observable = false; p.decision = "greeting is fine"; p.scenarios = nil }), []string{"unrecorded:"}},
		{"unrecorded other line", propReport(func(p *propertyReport) { p.observable = false; p.decision = "other"; p.scenarios = nil }), []string{"unrecorded:"}},
	}
	for _, c := range cases {
		var got []string
		for _, pr := range propEvidence().propertyProblems([]propertyReport{c.p}, req, "none") {
			got = append(got, pr.reason+":"+pr.scenario)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: problems %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPropertyCounts(t *testing.T) {
	parallel(t)
	ev := propEvidence()
	for name, c := range map[string]struct {
		p    propertyReport
		want int
	}{
		"in request":       {propReport(nil), 0},
		"only candidate":   {propReport(func(p *propertyReport) { p.request = "mul is memoized"; p.scenarios = nil }), 0},
		"blank request":    {propReport(func(p *propertyReport) { p.request = " "; p.scenarios = nil }), 0},
		"blank name":       {propReport(func(p *propertyReport) { p.property = ""; p.scenarios = nil }), 0},
		"in answers":       {propReport(func(p *propertyReport) { p.request = "answer: 1: a"; p.scenarios = nil }), 1},
		"in request, none": {propReport(func(p *propertyReport) { p.scenarios = nil }), 1},
	} {
		got := len(ev.propertyProblems([]propertyReport{c.p}, "make it fast", "Round 1:\n1. q?\n   answer: 1: a"))
		if got != c.want {
			t.Errorf("%s: %d problems, want %d", name, got, c.want)
		}
	}
}

func TestPropertyVerdictsAndText(t *testing.T) {
	parallel(t)
	props := []propertyReport{propReport(func(p *propertyReport) { p.property = "A\n a"; p.scenarios = nil }), propReport(func(p *propertyReport) { p.property = "B"; p.kind = "kept"; p.scenarios = nil })}
	problems := propEvidence().propertyProblems(props, "make it fast", "")
	if len(problems) != 2 {
		t.Fatalf("%d problems", len(problems))
	}
	if got, want := propertyConfirmLine(4, props[0], problems[0]), "4. property: A a (new) | request: make it fast | problem: no scenario exercises it"; got != want {
		t.Errorf("confirm line %q", got)
	}
	lines := applyPropertyVerdicts(props, problems, 3, []contractVerdict{{3, "confirmed", "yes\n sure"}, {3, "refuted", "ignored"}, {9, "confirmed", "range"}})
	if want := []string{"- A a (new) | request: make it fast | problem: no scenario exercises it | confirmed: yes sure"}; !reflect.DeepEqual(lines, want) {
		t.Errorf("lines %q", lines)
	}
	if problems[0].verdict != "confirmed" || problems[1].verdict != "refuted" || problems[1].confirmation != nil {
		t.Errorf("verdicts %+v", problems)
	}
	event := propertyEvent("h", props, []bool{true, true}, problems)
	probs := event["problems"].([]any)
	if event["event"] != "property_check" || event["package"] != "h" || probs[1].(map[string]any)["verdict"] != "refuted" || probs[1].(map[string]any)["confirmation"] != nil || probs[0].(map[string]any)["confirmation"] != "yes\n sure" {
		t.Errorf("event %v", event)
	}
	fb := contractFeedback(contractConfirmed{contradictions: []string{"- c"}, environment: []string{"- e"}, properties: []string{"- p1", "- p2"}})
	i, j, k := strings.Index(fb, "- c"), strings.Index(fb, "- e"), strings.Index(fb, "Property check: 2 problem(s)")
	if !(i < j && j < k) || !strings.HasSuffix(fb, "refused again.\n- p1\n- p2") {
		t.Errorf("feedback %q", fb)
	}
	if got := (contractConfirmed{contradictions: []string{"a"}, environment: []string{"b"}, properties: []string{"c"}}).all(); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("all %q", got)
	}
	if (contractConfirmed{properties: []string{"c"}}).empty() {
		t.Error("property problems are not empty")
	}
}

func TestPropertyPayloadValidation(t *testing.T) {
	parallel(t)
	const prop = `{"property":"p","request":"r","kind":"new","observable":true,"scenarios":["s"],"decision":"","issues":[{"problem":"missing","scenario":"s","quote":"q","why":"w"}]}`
	decodeFor := func(s string) map[string]any { return decodeReviewJSON(t, s) }
	for payload, valid := range map[string]bool{
		`{"findings":[],"environment":[],"properties":[]}`:                                                                            true,
		`{"findings":[],"environment":[],"properties":[` + prop + `]}`:                                                                true,
		`{"findings":[],"environment":[],"properties":{}}`:                                                                            false,
		`{"findings":[],"environment":[],"properties":["x"]}`:                                                                         false,
		`{"findings":[],"environment":[],"properties":[{"property":"p"}]}`:                                                            false,
		`{"findings":[],"environment":[],"properties":[` + strings.Replace(prop, `"kind":"new"`, `"kind":"old"`, 1) + `]}`:            false,
		`{"findings":[],"environment":[],"properties":[` + strings.Replace(prop, `"observable":true`, `"observable":"yes"`, 1) + `]}`: false,
		`{"findings":[],"environment":[],"properties":[` + strings.Replace(prop, `"scenarios":["s"]`, `"scenarios":[1]`, 1) + `]}`:    false,
		`{"findings":[],"environment":[],"properties":[` + strings.Replace(prop, `"missing"`, `"other"`, 1) + `]}`:                    false,
		`{"findings":[],"environment":[],"properties":[` + strings.Replace(prop, `"why":"w"`, `"why":"w","x":1`, 1) + `]}`:            false,
		`{"findings":[],"environment":[],"properties":[` + strings.Replace(prop, `"decision":"",`, ``, 1) + `]}`:                      false,
		`{"findings":[],"environment":[],"properties":[],"extra":1}`:                                                                  false,
		`{"findings":[],"environment":[]}`:                                                                                            true,
	} {
		_, _, props, why := parseContractReviewProperties(decodeFor(payload))
		if (why == "") != valid {
			t.Errorf("%s: reason %q, want valid=%v", payload, why, valid)
		}
		if valid && strings.Contains(payload, prop) && len(props) != 1 {
			t.Errorf("%s: %d properties", payload, len(props))
		}
	}
	_, _, _, why := parseContractReviewProperties(decodeFor(`{"findings":[],"environment":[],"properties":[],"extra":1}`))
	if why != `the payload must have exactly the fields "findings" and "environment", and optionally "properties"` {
		t.Errorf("reason %q", why)
	}
}

func decodeReviewJSON(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	d := json.NewDecoder(strings.NewReader(s))
	d.UseNumber()
	if err := d.Decode(&m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestContractCheckNumbersPropertyProblemsAfterContradictions(t *testing.T) {
	parallel(t)
	f := newShpFx(t, shpSubmit("c1", shpFiles()), ccCorrected("c2"), ccIntentOnly("c3"))
	f.setFeatureReport("corrected-report", ccCorrectedFeature, "FAILED", 1)
	prop := map[string]any{"property": "is_even is kept", "request": "calc.is_even(n)", "kind": "kept", "observable": true,
		"scenarios": []any{"No such scenario"}, "decision": "", "issues": []any{}}
	f.Prov.contract = []shpStep{
		shpAnswer(shpCall("r1", SubmitContractReviewTool, map[string]any{"findings": []any{ccContradiction()}, "environment": []any{}, "properties": []any{prop}})),
		ccConfirm("v1", ccVerdict(1, "refuted", "a correct implementation passes"), ccVerdict(2, "confirmed", "no scenario")),
	}
	exit, err := f.shape("Add calc.is_even(n).")
	if err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	reqs := f.Prov.contractRequests()
	shpContains(t, "review input", ccInput(reqs[0]), "\n\n# Request\n\nAdd calc.is_even(n).\n\n# Questions and answers\n\nnone")
	shpContains(t, "confirmation", ccInput(reqs[1]), "\n\n# Findings to verify\n\n1. scenario: Even numbers")
	shpContains(t, "confirmation", ccInput(reqs[1]), "\n2. property: is_even is kept (kept) | request: calc.is_even(n) | problem: no scenario exercises it")
	feedback := shpOutputs(f.Prov.requests()[1])["c1"]
	shpEq(t, "feedback", feedback, propertyHeadline(1)+"\n- is_even is kept (kept) | request: calc.is_even(n) | problem: no scenario exercises it | confirmed: no scenario")
	reviews := ccReviews(f)
	if len(reviews) != 2 || reviews[0]["result"] != "refused" || reviews[0]["evidenced"] != float64(1) || reviews[0]["confirmed"] != float64(0) {
		t.Fatalf("contract_review events %v", reviews)
	}
	checks := shpNamed(f.events(), "property_check")
	if len(checks) != 1 {
		t.Fatalf("property_check events %v", checks)
	}
	problems := checks[0]["problems"].([]any)
	if len(problems) != 1 || problems[0].(map[string]any)["reason"] != "uncovered" || problems[0].(map[string]any)["verdict"] != "confirmed" {
		t.Fatalf("property_check problems %v", problems)
	}
}
