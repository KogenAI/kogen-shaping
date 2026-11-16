package shape

import (
	"strings"
	"testing"
)

func TestExampleContradictionLine(t *testing.T) {
	for _, tc := range []struct {
		scenario, expected, derived, why, want string
	}{
		{"Log outcome", `{"http_status":204,"code":299}`, `{"code":299}`, "the rules never state http_status", `Log outcome: differs in http_status | expected {"http_status":204,"code":299} | rules give {"code":299} | why: the rules never state http_status`},
		{"Retry", `{"state":"completed","code":503}`, `{"state":"errored","code":503,"retries":0}`, "state and retries differ", `Retry: differs in retries, state | expected {"state":"completed","code":503} | rules give {"state":"errored","code":503,"retries":0} | why: state and retries differ`},
		{"Order", `["a","b"]`, `["b","a"]`, "order matters", `Order: expected ["a","b"], rules give ["b","a"] | why: order matters`},
		{"Wrapped", `{"a":1}`, `{"a":2}`, " first\n\t second ", `Wrapped: differs in a | expected {"a":1} | rules give {"a":2} | why: first second`},
		{"Empty", `{"a":1}`, `{"a":2}`, "   \n", `Empty: differs in a | expected {"a":1} | rules give {"a":2}`},
		{"Invalid", `{`, `{"a":2}`, "", `Invalid: expected {, rules give {"a":2}`},
		{"Scalar", `1`, `2`, "", `Scalar: expected 1, rules give 2`},
		{"Semantic", `{"number":1,"nested":{"a":1,"b":2},"missing":null}`, `{"nested":{"b":2,"a":1},"number":1.0}`, "", `Semantic: differs in missing | expected {"number":1,"nested":{"a":1,"b":2},"missing":null} | rules give {"nested":{"b":2,"a":1},"number":1.0}`},
		{"Members", `a:1,b:2`, `b:3,a:1`, "", `Members: differs in b | expected a:1,b:2 | rules give b:3,a:1`},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if got := exampleContradictionLine(tc.scenario, tc.expected, tc.derived, tc.why); got != tc.want {
				t.Errorf("got %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestExampleContradictionLineCut(t *testing.T) {
	const scenario = "Log HTTP outcomes without retries — code 299"
	const expected = `{"http_status":204,"posts":1,"deliveries":1,"payload_matches":true,"signed":true,"state":"completed","code":299,"error_present":false,"signature_hidden":true,"html_has_outcome":true}`
	const derived = `{"posts":1,"deliveries":1,"payload_matches":true,"signed":true,"state":"completed","code":299,"error_present":false,"signature_hidden":true,"html_has_outcome":true}`
	const why = "the rules do not state the http status of the replay"
	line := exampleContradictionLine(scenario, expected, derived, why)
	cut := cutProblems([]string{line})[0]
	if len([]rune(cut)) > 300 || !strings.Contains(cut, " ... ") {
		t.Fatalf("line was not bounded and shortened: %q", cut)
	}
	if !strings.HasPrefix(cut, scenario+`: differs in http_status | expected {"http_status":204`) {
		t.Errorf("cut lost differing field: %q", cut)
	}
	if !strings.HasSuffix(cut, " | why: "+why) {
		t.Errorf("cut lost reason: %q", cut)
	}
}
