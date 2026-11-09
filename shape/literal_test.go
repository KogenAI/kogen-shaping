package shape

import (
	"slices"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

const (
	litRequest = "Register /demo_one and /demo-two.\nSign in with password: hunter2-demo-secret."
	litQuote   = "Register /demo_one and /demo-two."
)

func litItem(literal, source, quote, status, packageValue string) literalItem {
	return literalItem{literal: literal, source: source, quote: quote, status: status, packageValue: packageValue, why: "w"}
}

func TestLiteralOutcomeRules(t *testing.T) {
	parallel(t)
	files := []packageFile{{"intent.md", "registers /demo_one here"}, {"acceptance.feature", "uses /demo-one there and 7443"}}
	sources := map[string]string{"request": litRequest, "answer 1": "port 7443", "round 1 answer 1": "port 7443"}
	red := verify.NewRedactor([]string{"MY_TOKEN=tok-9f8e7d"}, "")
	cases := []struct {
		name   string
		item   literalItem
		status string
	}{
		{"present", litItem("/demo_one", "request", litQuote, "required", ""), "present"},
		{"present in a later file", litItem("7443", "answer 1", "port 7443", "required", ""), "present"},
		{"present from an earlier round", litItem("7443", "round 1 answer 1", "port 7443", "required", ""), "present"},
		{"missing", litItem("/demo-two", "request", litQuote, "required", ""), "missing"},
		{"present despite a stated substitute", litItem("/demo_one", "request", litQuote, "required", "/demo-one"), "present"},
		{"substituted", litItem("/demo-two", "request", litQuote, "required", "/demo-one"), "changed"},
		{"substitute not in the package", litItem("/demo-two", "request", litQuote, "required", "/demo_two"), "missing"},
		{"substitute equal to the literal", litItem("/demo-two", "request", litQuote, "required", "/demo-two"), "missing"},
		{"superseded", litItem("/demo-two", "request", litQuote, "superseded", ""), "superseded"},
		{"informal", litItem("/demo-two", "request", litQuote, "informal", ""), "informal"},
		{"secret by status", litItem("", "request", "", "secret", ""), "secret"},
		{"secret by the redactor on the literal", litItem("tok-9f8e7d", "request", "tok-9f8e7d", "required", ""), "secret"},
		{"secret by the redactor on the source", litItem("hunter2-demo-secret", "request", "password: hunter2-demo-secret", "required", ""), "secret"},
		{"blank literal", litItem(" ", "request", litQuote, "required", ""), "unverified"},
		{"unknown source", litItem("/demo-two", "answer 2", litQuote, "required", ""), "unverified"},
		{"quote not in the source", litItem("/demo-two", "request", "Register /demo-two.", "required", ""), "unverified"},
		{"literal not in its quote", litItem("/demo_three", "request", litQuote, "required", ""), "unverified"},
		{"case differs", litItem("/Demo_one", "request", litQuote, "required", ""), "unverified"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parallel(t)
			event, _ := literalOutcome([]literalItem{tc.item}, sources, files, red)
			got := event["items"].([]any)[0].(map[string]any)["status"]
			shpEq(t, "status", got.(string), tc.status)
			shpEq(t, "count", event[tc.status], 1)
		})
	}

	t.Run("feedback, event and redaction", func(t *testing.T) {
		parallel(t)
		missing := litItem("/demo-two", "request", litQuote, "required", "")
		secret := litItem("hunter2-demo-secret", "request", "password: hunter2-demo-secret", "required", "")
		leaky := litItem("/demo-two", "request", litQuote, "required", "")
		leaky.why = "see tok-9f8e7d"
		event, lines := literalOutcome([]literalItem{missing, secret, leaky}, sources, files, red)
		want := []string{
			"Literal check: 2 required literal values are not carried exactly by the package; copy each value byte for byte into the Intent or its acceptance where the Intent uses it (descriptive text may stay paraphrased), or restore the substituted value:",
			`literal: /demo-two | source: request | source_quote: "Register /demo_one and /demo-two." | status: missing | why: w`,
			`literal: /demo-two | source: request | source_quote: "Register /demo_one and /demo-two." | status: missing | why: see [REDACTED]`,
		}
		if !slices.Equal(lines, want) {
			t.Fatalf("lines\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
		}
		shpEq(t, "event", event["event"].(string), "literal_check")
		shpEq(t, "compared", event["status"].(string), "compared")
		shpEq(t, "missing", event["missing"], 2)
		shpEq(t, "secret", event["secret"], 1)
		items := event["items"].([]any)
		shpEq(t, "secret item", len(items[1].(map[string]any)), 2)
		if strings.Contains(strings.Join(lines, "\n")+litAny(event), "hunter2-demo-secret") || strings.Contains(litAny(event), "tok-9f8e7d") {
			t.Fatalf("secret exposed: %v %v", lines, event)
		}

		sub := litItem("/demo-two", "request", litQuote, "required", "/demo-one")
		_, lines = literalOutcome([]literalItem{sub}, sources, files, red)
		shpEq(t, "changed line", lines[1], `literal: /demo-two | source: request | source_quote: "Register /demo_one and /demo-two." | status: changed | package_value: /demo-one | why: w`)

		event, lines = literalOutcome([]literalItem{litItem("/demo_one", "request", litQuote, "required", "")}, sources, files, red)
		if lines != nil {
			t.Fatalf("lines %q", lines)
		}
		shpEq(t, "present", event["present"], 1)
	})
}

func litAny(v any) string {
	var b strings.Builder
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case string:
			b.WriteString(x + "\n")
		case []any:
			for _, e := range x {
				walk(e)
			}
		case map[string]any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	walk(v)
	return b.String()
}

func TestLiteralCheckInput(t *testing.T) {
	parallel(t)
	rounds := []map[string]any{
		{"questions": []any{
			map[string]any{"question": "Q1", "options": []any{map[string]any{"label": "alpha"}, map[string]any{"label": "beta"}}},
			map[string]any{"question": "Q2"}},
			"answers": map[string]any{"1": "b", "2": "typed words"}},
		{"questions": []any{
			map[string]any{"question": "Q3", "options": []any{map[string]any{"label": "gamma"}}},
			map[string]any{"question": "Q4"},
			map[string]any{"question": "Q5"}},
			"answers": map[string]any{"1": "a", "2": "z", "3": ""}},
	}
	sources := literalSources("the request", rounds)
	want := map[string]string{"request": "the request", "round 1 answer 1": "beta", "round 1 answer 2": "typed words", "answer 1": "gamma", "answer 2": "z"}
	if len(sources) != len(want) {
		t.Fatalf("sources %v", sources)
	}
	for k, v := range want {
		shpEq(t, k, sources[k], v)
	}
	shpEq(t, "answers", literalAnswersText(rounds), "round 1 answer 1: beta\nround 1 answer 2: typed words\nanswer 1: gamma\nanswer 2: z")
	shpEq(t, "no answers", literalAnswersText(nil), "(none)")
	shpEq(t, "input", literalInput("s-lug", "req", nil, []packageFile{{"a.md", "A"}, {"b.feature", "B"}}),
		"Slug: s-lug\n\n# Request\n\nreq\n\n# Answers\n\n(none)\n\n# Candidate package\n\n## a.md\n\nA\n\n## b.feature\n\nB\n\n")
}

func TestLiteralCandidatePackageOrder(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	const slug = "pkg-order"
	intent := "# T\n\nSlug: " + slug + "\nModel: .kogen/intents/" + slug + "/model.qnt invariants=a seed=1 samples=1 steps=1\nAcceptance:\n- features/steps/b__x.js\n- .kogen/intents/" + slug + "/acceptance.feature\n- features/steps/a__x.js\n- features/steps/b__x.js\n"
	for p, c := range map[string]string{".kogen/intents/" + slug + "/intent.md": intent, ".kogen/intents/" + slug + "/acceptance.feature": "F", ".kogen/intents/" + slug + "/model.qnt": "M",
		"features/steps/a__x.js": "A", "features/steps/b__x.js": "B"} {
		f.write(p, c)
	}
	sh := &shaping{root: f.Root, slug: slug}
	files, err := sh.candidatePackage()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range files {
		got = append(got, p.path)
	}
	shpEq(t, "order", strings.Join(got, ","), ".kogen/intents/"+slug+"/intent.md,.kogen/intents/"+slug+"/acceptance.feature,.kogen/intents/"+slug+"/model.qnt,features/steps/b__x.js,features/steps/a__x.js")
}

func TestParseLiteralCheck(t *testing.T) {
	parallel(t)
	good := func() map[string]any {
		return map[string]any{"literal": "l", "source": "request", "source_quote": "q", "package_value": "", "why": "w", "status": "required"}
	}
	with := func(k string, v any) map[string]any { m := good(); m[k] = v; return m }
	without := func(k string) map[string]any { m := good(); delete(m, k); return m }
	items, err := parseLiteralCheck(map[string]any{"items": []any{good()}})
	if err != nil || len(items) != 1 || items[0].literal != "l" || items[0].status != "required" {
		t.Fatalf("items %v, err %v", items, err)
	}
	if items, err = parseLiteralCheck(map[string]any{"items": []any{}}); err != nil || len(items) != 0 {
		t.Fatalf("items %v, err %v", items, err)
	}
	for name, args := range map[string]map[string]any{
		"items missing":      {},
		"items not a list":   {"items": "x"},
		"an extra payload":   {"items": []any{}, "more": 1},
		"item not an object": {"items": []any{"x"}},
		"missing field":      {"items": []any{without("why")}},
		"mistyped field":     {"items": []any{with("literal", 7)}},
		"extra item field":   {"items": []any{with("extra", "x")}},
		"unknown status":     {"items": []any{with("status", "maybe")}},
		"mistyped status":    {"items": []any{with("status", nil)}},
	} {
		_, err := parseLiteralCheck(args)
		e, ok := sys.AsError(err)
		if !ok || e.Class != "shape" || e.Reason != "literal_check_invalid" || e.Exit != 1 || !strings.Contains(e.Error(), "the literal check answer is not a valid submit_literal_check payload (") || !strings.HasSuffix(e.Error(), "); nothing was published") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestLiteralInstructions(t *testing.T) {
	parallel(t)
	lines := strings.Split(literalInstructions, "\n")
	for _, want := range []string{
		"A required literal is a value that the request or a settled answer explicitly requires the implementation or an example to use exactly, such as a path, command, name, identifier, number or example input; descriptive text that may be paraphrased is not a literal.",
		"Give status superseded for a value that the request or a later answer explicitly replaces, and status secret for secret material such as a password, token, key or credential; for secret material leave literal, source_quote, package_value and why empty.",
		"Copy each literal exactly as its source writes it, with source_quote copied exactly from that source and containing the literal; Kogen itself compares each literal with the package byte for byte.",
		"Give status informal for a value that appears only in informal background, an aside or an example the source marks optional; Kogen never demands an informal value.",
	} {
		if !slices.Contains(lines, want) {
			t.Errorf("instructions lack the line %q", want)
		}
	}
	if !strings.HasSuffix(literalInstructions, "\nAnswer by calling submit_literal_check.") {
		t.Errorf("instructions end %q", literalInstructions[len(literalInstructions)-60:])
	}
}

// TestExternalShapeLiteralCheck: an external shaper runs the check as one fresh session per
// checked submission with the check schema, refuses on a missing literal and records the event.
func TestExternalShapeLiteralCheck(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	const request = "Add calc.is_even(n). Raise exactly ZeroDivisionError for 0."
	const quote = "Raise exactly ZeroDivisionError for 0."
	item := func(status string) func(int, string) (executor.RoundResult, error) {
		return result(map[string]any{"items": []any{map[string]any{"literal": "ZeroDivisionError", "source": "request", "source_quote": quote,
			"package_value": "", "why": "because", "status": status}}})
	}
	ack := map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}}
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(shpFiles()), result(shpFiles()), result(ack)},
		literal: []func(int, string) (executor.RoundResult, error){item("required"), item("informal")}}
	f.shapeWith(x, request)
	if len(x.literalStarts) != 3 {
		t.Fatalf("%d literal sessions", len(x.literalStarts))
	}
	for i, s := range x.literalStarts {
		shpContains(t, "task", s.Brief.Stable, literalInstructions)
		shpContains(t, "task", s.Brief.Stable, "# Request\n\n"+request+"\n\n# Answers\n\n(none)\n\n# Candidate package\n\n## .kogen/intents/is-even/intent.md\n\n")
		shpContains(t, "dir", s.Dir, "executor-literal-"+string(rune('1'+i)))
		shpContains(t, "schema", string(s.Schema), `"source_quote"`)
	}
	var feedback string
	for _, in := range x.inputs {
		feedback += in.Feedback
	}
	shpContains(t, "feedback", feedback, "Literal check: 1 required literal values are not carried exactly by the package;")
	shpContains(t, "feedback line", feedback, `literal: ZeroDivisionError | source: request | source_quote: "`+quote+`" | status: missing | why: because`)
	events := shpNamed(f.events(), "literal_check")
	if len(events) != 3 || events[0]["missing"] != float64(1) || events[1]["informal"] != float64(1) {
		t.Fatalf("events %v", events)
	}
	noNativeRequests(t, f)
}

// Preparation instructions must not become product obligations. Identical names
// remain exact literals when the requested product actually exposes them.
func TestLiteralInstructionsSeparatePreparationFromProduct(t *testing.T) {
	parallel(t)
	for _, want := range []string{"not the procedure used to shape the Intent", "give it status informal", "If the requested product itself exposes that same name or value, it remains required", "add a probe parameter named capture_output"} {
		if !strings.Contains(literalInstructions, want) {
			t.Errorf("literal boundary missing %q", want)
		}
	}
}
