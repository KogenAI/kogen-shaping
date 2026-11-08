package shape

import "testing"

func TestExampleVerdict(t *testing.T) {
	const expected = `"status":0,"out":"61610a0a"`
	for _, tc := range []struct{ expected, derived, verdict string }{
		{expected, `{"status":0,"out":"61610a0a"}`, "follows"},
		{expected, `status:0, out:61610a0a`, "follows"},
		{expected, `{"out":"61610a0a","status":0}`, "follows"},
		{expected, `{"status":1,"out":"61610a0a"}`, "contradicts"},
		{expected, ``, "unsupported"},
		{`"a:b"`, `"a\u003ab"`, "follows"},
		{`["a:b"]`, `["a\u003ab"]`, "follows"},
		{`{"out":"a,b"}`, `{"out":"ab"}`, "contradicts"},
		{`{"out":[1,2]}`, `{"out":[2,1]}`, "contradicts"},
	} {
		if got := exampleVerdict(tc.expected, tc.derived, "contradicts"); got != tc.verdict {
			t.Errorf("%s vs %s: got %s want %s", tc.expected, tc.derived, got, tc.verdict)
		}
	}
}
