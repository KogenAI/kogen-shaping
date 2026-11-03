package shape

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
)

// qcPayload is a question check answer with the given items.
func qcPayload(items ...any) map[string]any { return map[string]any{"items": items} }

func qcItem(number any, class string) map[string]any {
	return map[string]any{"number": number, "class": class, "why": "w"}
}

func TestParseQuestionCheck(t *testing.T) {
	parallel(t)
	cases := []struct {
		name string
		args map[string]any
		n    int
		want string // the invalid reason; "" for a valid answer
	}{
		{"native int64, in order", qcPayload(qcItem(int64(1), "ui_ux"), qcItem(int64(2), "technical")), 2, ""},
		{"external float64, any order", qcPayload(qcItem(float64(2), "handoff"), qcItem(float64(1), "environment")), 2, ""},
		{"json.Number", qcPayload(qcItem(json.Number("1"), "answered")), 1, ""},
		{"every class", qcPayload(qcItem(1, "ui_ux"), qcItem(2, "handoff"), qcItem(3, "answered"), qcItem(4, "technical"), qcItem(5, "product_policy"), qcItem(6, "environment"), qcItem(7, "repository_fact")), 7, ""},
		{"items missing", map[string]any{}, 1, "items must be a list"},
		{"items not a list", map[string]any{"items": "ui_ux"}, 1, "items must be a list"},
		{"item not an object", qcPayload("ui_ux"), 1, "an item is malformed"},
		{"number not an integer", qcPayload(qcItem(1.5, "ui_ux")), 1, "an item is malformed"},
		{"number a string", qcPayload(qcItem("1", "ui_ux")), 1, "an item is malformed"},
		{"unknown class", qcPayload(qcItem(1, "design")), 1, "an item is malformed"},
		{"why not a string", qcPayload(map[string]any{"number": 1, "class": "ui_ux", "why": 3}), 1, "an item is malformed"},
		{"why missing", qcPayload(map[string]any{"number": 1, "class": "ui_ux"}), 1, "an item is malformed"},
		{"too few", qcPayload(qcItem(1, "ui_ux")), 2, "expected one item for each question 1 to 2"},
		{"too many", qcPayload(qcItem(1, "ui_ux"), qcItem(2, "ui_ux")), 1, "expected one item for each question 1 to 1"},
		{"duplicate number", qcPayload(qcItem(1, "ui_ux"), qcItem(1, "technical")), 2, "expected one item for each question 1 to 2"},
		{"number out of range", qcPayload(qcItem(0, "ui_ux"), qcItem(1, "ui_ux")), 2, "expected one item for each question 1 to 2"},
	}
	for _, c := range cases {
		items, err := parseQuestionCheck(c.args, c.n)
		if c.want == "" {
			if err != nil || len(items) != c.n {
				t.Errorf("%s: %v, %d items", c.name, err, len(items))
			}
			for i, it := range items {
				if it.number != i+1 {
					t.Errorf("%s: items not in question order: %v", c.name, items)
				}
			}
			continue
		}
		want := "shape/question_check_invalid: the question check answer is not a valid submit_question_check payload (" + c.want + "); nothing was asked"
		if err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %s", c.name, err, want)
		}
	}
}

func TestEvidenceMarkers(t *testing.T) {
	parallel(t)
	citations := []struct {
		line string
		want []string
	}{
		{"- A (source: request)", nil},
		{"- A (evidence: lib/a.rb) (source: request)", []string{"lib/a.rb"}},
		{"- A (evidence: lib/a.rb; ran `ls -l`) (source: answer 2)", []string{"lib/a.rb; ran `ls -l`"}},
		{"- A (evidence: lib/a.rb) and (evidence: ran `echo (x)`) (source: request)", []string{"lib/a.rb", "ran `echo (x)`"}},
		{"- A (evidence: lib/a.rb", []string{"lib/a.rb"}},
		{"- A (evidence: ) (source: request)", []string{""}},
	}
	for _, c := range citations {
		if got := evidenceCitations(c.line); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: citations %q, want %q", c.line, got, c.want)
		}
	}
	if got := splitCitation("a; ran `x; y`; `b/`; ; c"); !reflect.DeepEqual(got, []string{"a", "ran `x; y`", "`b/`", "", "c"}) {
		t.Errorf("split: %q", got)
	}
	items := map[string]evidenceItem{
		"lib/a.rb":        {path: "lib/a.rb"},
		"`lib/a.rb`":      {path: "lib/a.rb"},
		"lib/":            {path: "lib/"},
		"ran `ls; pwd`":   {command: "ls; pwd"},
		"":                {empty: true},
		" ":               {empty: true},
		"``":              {empty: true},
		"ran ``":          {command: "", empty: true},
		"ran ls":          {path: "ran ls"},
		"ran `a` and `b`": {command: "a` and `b"},
	}
	for in, want := range items {
		if got := parseEvidenceItem(in); got != want {
			t.Errorf("item %q: %+v, want %+v", in, got, want)
		}
	}
}

// qcChecks are the question check requests among a provider's requests.
func qcChecks(reqs []map[string]any) []map[string]any {
	var out []map[string]any
	for _, r := range reqs {
		if len(shpToolNames(r)) == 1 && shpToolNames(r)[0] == SubmitQuestionCheckTool {
			out = append(out, r)
		}
	}
	return out
}

// qcText is the text of a request's one user message.
func qcText(req map[string]any) string {
	input, _ := req["input"].([]any)
	m, _ := input[0].(map[string]any)
	s, _ := m["content"].(string)
	return s
}

const qcToolJSON = `{"type":"function","name":"submit_question_check","strict":true,"description":"Submit the question check: one item per question with its number, class and reason.","parameters":{"type":"object","properties":{"items":{"type":"array","items":{"type":"object","properties":{"number":{"type":"integer"},"class":{"type":"string","enum":["ui_ux","handoff","answered","technical","product_policy","environment","repository_fact"]},"why":{"type":"string"}},"required":["number","class","why"],"additionalProperties":false}}},"required":["items"],"additionalProperties":false}}`

func TestShapeQuestionCheckRequestAndPublication(t *testing.T) {
	parallel(t)
	f := newShpFx(t,
		shpSubmit("q1", map[string]any{"questions": []any{
			map[string]any{"question": "Sign in to the provider", "kind": "handoff"},
			shpQuestion("Which wording?", "keep", "shorter"),
		}}),
		shpQuestionCheck("c1", "handoff", "ui_ux"),
		shpSubmit("q2", map[string]any{"questions": []any{shpQuestion("Which icon?", "star", "heart")}}),
		shpUI(1))
	exit, err := f.shape("Add calc.is_even(n).")
	if err != nil || exit != 5 {
		t.Fatalf("exit %d, err %v", exit, err)
	}
	shpContains(t, "stdout", f.Stdout.String(), "2. Which wording?\n   a) keep (recommended)\n   b) shorter\n")
	if !f.exists(".kogen/local/shaping/is-even/questions.json") {
		t.Fatal("questions.json is written for questions that reach the user")
	}
	exit, err = f.shape("1: done\n2: b")
	if err != nil || exit != 5 {
		t.Fatalf("exit %d, err %v", exit, err)
	}
	checks := qcChecks(f.Prov.requests())
	shpEq(t, "question checks", len(checks), 2)
	first := checks[0]
	shpEq(t, "instructions", first["instructions"].(string), questionInstructions)
	if !strings.HasPrefix(questionInstructions, "You check the questions a shaper wants to ask the user about one feature request.") ||
		!strings.HasSuffix(questionInstructions, "\nAnswer by calling submit_question_check.") {
		t.Fatalf("instructions %q", questionInstructions)
	}
	var want, got any
	if err := json.Unmarshal([]byte(qcToolJSON), &want); err != nil {
		t.Fatal(err)
	}
	got = first["tools"].([]any)[0]
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tool %v, want %v", got, want)
	}
	shpEq(t, "first input", qcText(first), "Slug: is-even\n\n# Request\n\nAdd calc.is_even(n).\n\n# Earlier questions and answers\n\nnone\n\n# Questions\n\n"+
		"1. Sign in to the provider\n   kind: handoff\n2. Which wording?\n   a) keep (recommended)\n   b) shorter")
	shpEq(t, "second input", qcText(checks[1]), "Slug: is-even\n\n# Request\n\nAdd calc.is_even(n).\n\n# Earlier questions and answers\n\n"+
		"Round 1:\n1. Sign in to the provider\n   answer: 1: done\n2. Which wording?\n   a) keep\n   b) shorter\n   answer: 2: b\n\n# Questions\n\n"+
		"1. Which icon?\n   a) star (recommended)\n   b) heart")
	if first["prompt_cache_key"] == nil || first["prompt_cache_key"] == f.Prov.requests()[0]["prompt_cache_key"] {
		t.Fatal("the question check has its own conversation identity")
	}
	events := shpNamed(f.events(), "question_check")
	shpEq(t, "events", len(events), 2)
	if b, _ := json.Marshal(events[0]); string(b) != `{"asked":2,"event":"question_check","items":[{"class":"handoff","number":1,"question":"Sign in to the provider","why":"a handoff question"},{"class":"ui_ux","number":2,"question":"Which wording?","why":"a ui_ux question"}],"refused":0}` {
		t.Fatalf("event %s", b)
	}
	if len(shpNamed(f.events(), "submission_refused")) != 0 {
		t.Fatal("nothing is refused when every question may reach the user")
	}
}

const qcHeadline = "Question check: %s question(s) are not UI/UX choices left to the user; nothing was asked. Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff."

func TestShapeQuestionCheckRefusesTheWholeSubmission(t *testing.T) {
	parallel(t)
	qs := []any{shpQuestion("Which wording?", "keep", "shorter"), shpQuestion("Which engine?"), shpQuestion("Settled?"), shpQuestion("Charge past-due?"), shpQuestion("Is there a stub?"), shpQuestion("Is the CLI installed?")}
	f := newShpFx(t,
		shpSubmit("q1", map[string]any{"questions": qs}),
		shpQuestionCheck("c1", "ui_ux", "technical", "answered", "product_policy", "repository_fact", "environment"),
		shpSubmit("q2", map[string]any{"questions": qs[:1]}),
		shpUI(1))
	exit, err := f.shape("Add calc.is_even(n).")
	if err != nil || exit != 5 {
		t.Fatalf("exit %d, err %v", exit, err)
	}
	reqs := shpShaperRequests(f.Prov.requests())
	wantFeedback := strings.Join([]string{
		strings.Replace(qcHeadline, "%s", "5 of 6", 1),
		"- question 2 (technical): Which engine? | why: a technical question | decide it from what the repository and the request establish",
		"- question 3 (answered): Settled? | why: a answered question | the request or an earlier answer settles it; follow it and keep its (source: request) or (source: answer N)",
		"- question 4 (product_policy): Charge past-due? | why: a product_policy question | decide it from what the repository and the request establish",
		"- question 5 (repository_fact): Is there a stub? | why: a repository_fact question | decide it from what the repository and the request establish",
		"- question 6 (environment): Is the CLI installed? | why: a environment question | do not ask about the environment; proceed without it and record that Decision, or, if the request cannot be shaped without it, submit only such environment questions again and Shape ends with an environment fault",
	}, "\n")
	shpEq(t, "feedback", shpOutputs(reqs[1])["q1"], wantFeedback)
	// The refusal saved no round and printed nothing; the UI/UX question went through afterwards.
	shpEq(t, "stdout", f.Stdout.String(), "Questions for is-even:\n1. Which wording?\n   a) keep (recommended)\n   b) shorter\n"+
		"Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\\n' | kogen intent shape is-even -\nOr enter all: recommended to use every question's recommended option.\n")
	var state struct {
		Rounds []map[string]any `json:"rounds"`
	}
	if err := json.Unmarshal([]byte(f.read(".kogen/local/shaping/is-even/state.json")), &state); err != nil || len(state.Rounds) != 1 {
		t.Fatalf("%d rounds, err %v: a refused submission saves no round", len(state.Rounds), err)
	}
	refused := shpNamed(f.events(), "submission_refused")
	shpEq(t, "refusals", len(refused), 1)
	shpEq(t, "gate", refused[0]["gate"].(string), "question_check")
	shpEq(t, "problem lines", len(refused[0]["problems"].([]any)), 5)
	check := shpNamed(f.events(), "question_check")[0]
	shpEq(t, "asked", check["asked"].(float64), 1)
	shpEq(t, "refused", check["refused"].(float64), 5)
}

func TestShapeQuestionRefusalsCountAsInvalidSubmissions(t *testing.T) {
	parallel(t)
	var steps []shpStep
	for _, id := range []string{"a", "b", "c"} {
		steps = append(steps, shpSubmit("q"+id, map[string]any{"questions": []any{shpQuestion("Which engine?")}}), shpQuestionCheck("c"+id, "technical"))
	}
	f := newShpFx(t, steps...)
	exit, err := f.shape("Add calc.is_even(n).")
	e := shpWantError(t, err, "shape", "invalid_output", "after 3 attempts", 1)
	shpEq(t, "exit", exit, 1)
	shpEq(t, "lines", len(e.Lines), 1)
	if f.exists(".kogen/local/shaping/is-even/questions.json") || f.Stdout.Len() != 0 {
		t.Fatal("nothing is asked")
	}
	shpEq(t, "refusals", len(shpNamed(f.events(), "submission_refused")), 3)
}

func TestShapeEnvironmentQuestions(t *testing.T) {
	parallel(t)
	env := map[string]any{"questions": []any{shpQuestion("Is a key set?", "yes", "no"), shpQuestion("Is the CLI installed?")}}
	envCheck := func(id string) shpStep { return shpQuestionCheck(id, "environment", "environment") }
	faultLines := []string{"- Is a key set?", "- Is the CLI installed?"}
	fault := func(t *testing.T, f *shpFx, exit int, err error) {
		t.Helper()
		e := shpWantError(t, err, "environment", "shape_environment", "", 3)
		shpEq(t, "message", e.Error(), "environment/shape_environment: shaping cannot go on without something only your environment can provide; nothing was asked or published")
		shpEq(t, "lines", strings.Join(e.Lines, "\n"), strings.Join(faultLines, "\n"))
		shpEq(t, "exit", exit, 3)
		if f.exists(".kogen/local/shaping/is-even/questions.json") || f.exists(".kogen/local/shaping/is-even/state.json") || f.exists(".kogen/intents/is-even") {
			t.Fatal("nothing is asked or published")
		}
	}

	t.Run("a first environment-only refusal is feedback, then the shaper proceeds", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t, shpSubmit("q1", env), envCheck("c1"), shpSubmit("c2", shpFiles()), shpAck("c3"))
		if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
			t.Fatalf("exit %d, err %v", exit, err)
		}
		shpContains(t, "feedback", shpOutputs(shpShaperRequests(f.Prov.requests())[1])["q1"], "(environment): Is a key set? | why: a environment question | do not ask about the environment;")
	})
	t.Run("the second environment-only refusal ends Shape", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t, shpSubmit("q1", env), envCheck("c1"), shpSubmit("q2", env), envCheck("c2"))
		exit, err := f.shape("Add calc.is_even(n).")
		fault(t, f, exit, err)
		shpEq(t, "refusals", len(shpNamed(f.events(), "submission_refused")), 2)
	})
	t.Run("an environment-only refusal at the last allowed submission ends Shape", func(t *testing.T) {
		parallel(t)
		both := map[string]any{"questions": []any{shpQuestion("Q?")}, "files": shpFiles()["files"]}
		f := newShpFx(t, shpSubmit("b1", both), shpSubmit("b2", both), shpSubmit("q1", env), envCheck("c1"))
		exit, err := f.shape("Add calc.is_even(n).")
		fault(t, f, exit, err)
		refused := shpNamed(f.events(), "submission_refused")
		shpEq(t, "refusals", len(refused), 3)
		shpEq(t, "last gate", refused[2]["gate"].(string), "question_check")
	})
	t.Run("another class in the submission is no environment fault", func(t *testing.T) {
		parallel(t)
		both := map[string]any{"questions": []any{shpQuestion("Q?")}, "files": shpFiles()["files"]}
		f := newShpFx(t, shpSubmit("b1", both), shpSubmit("b2", both), shpSubmit("q1", env), shpQuestionCheck("c1", "environment", "technical"))
		exit, err := f.shape("Add calc.is_even(n).")
		shpWantError(t, err, "shape", "invalid_output", "", 1)
		shpEq(t, "exit", exit, 1)
	})
	t.Run("an environment-only refusal after a mixed one is the first", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t, shpSubmit("q1", env), shpQuestionCheck("c1", "environment", "technical"), shpSubmit("q2", env), envCheck("c2"), shpSubmit("c3", shpFiles()), shpAck("c4"))
		if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
			t.Fatalf("exit %d, err %v", exit, err)
		}
	})
}

func TestShapeQuestionCheckInvalidAnswers(t *testing.T) {
	parallel(t)
	ask := map[string]any{"questions": []any{shpQuestion("Which wording?", "keep", "shorter"), shpQuestion("Which icon?")}}
	cases := []struct {
		name string
		step shpStep
		why  string
	}{
		{"no call", shpAnswer(shpCall("c1", "probe", map[string]any{"command": "true"})), "expected one submit_question_check call"},
		{"two calls", shpAnswer(shpCall("c1", "submit_question_check", qcPayload(qcItem(1, "ui_ux"), qcItem(2, "ui_ux"))), shpCall("c2", "submit_question_check", qcPayload())), "expected one submit_question_check call"},
		{"items not a list", shpAnswer(shpCall("c1", "submit_question_check", map[string]any{"items": 3})), "items must be a list"},
		{"malformed item", shpAnswer(shpCall("c1", "submit_question_check", qcPayload(qcItem(1, "ui_ux"), qcItem(2, "layout")))), "an item is malformed"},
		{"a question without an item", shpAnswer(shpCall("c1", "submit_question_check", qcPayload(qcItem(1, "ui_ux")))), "expected one item for each question 1 to 2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parallel(t)
			f := newShpFx(t, shpSubmit("q1", ask), c.step)
			exit, err := f.shape("Add calc.is_even(n).")
			shpWantError(t, err, "shape", "question_check_invalid", "", 1)
			shpEq(t, "message", err.Error(), "shape/question_check_invalid: the question check answer is not a valid submit_question_check payload ("+c.why+"); nothing was asked")
			shpEq(t, "exit", exit, 1)
			if f.exists(".kogen/local/shaping/is-even/questions.json") || f.exists(".kogen/local/shaping/is-even/state.json") || f.Stdout.Len() != 0 {
				t.Fatal("nothing is asked or saved")
			}
		})
	}
}

func TestExternalShapeQuestionCheck(t *testing.T) {
	parallel(t)
	ask := map[string]any{"questions": []any{shpQuestion("Which wording?", "keep", "shorter")}}
	check := func(class string) func(int, string) (executor.RoundResult, error) {
		return result(map[string]any{"items": []any{map[string]any{"number": float64(1), "class": class, "why": "w"}}})
	}
	t.Run("a session of its own: text, schema, directory, rounds and feedback", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(ask), check("technical"), result(shpFiles()), result(map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}})}}
		exit, err := f.shapeWith(x, "Add calc.is_even(n).")
		if err != nil || exit != 0 {
			t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
		}
		shpEq(t, "sessions", len(x.starts), 2)
		task := x.starts[1]
		if !strings.HasSuffix(task.Dir, "/executor-question-1") || task.Role != executor.RoleShape {
			t.Fatalf("task %+v", task)
		}
		shpEq(t, "text", task.Brief.Stable, questionInstructions+"\n\n"+"Slug: is-even\n\n# Request\n\nAdd calc.is_even(n).\n\n# Earlier questions and answers\n\nnone\n\n# Questions\n\n1. Which wording?\n   a) keep (recommended)\n   b) shorter")
		shpEq(t, "dynamic", task.Brief.Dynamic, "")
		var schema, want map[string]any
		if json.Unmarshal(task.Schema, &schema) != nil || json.Unmarshal([]byte(qcToolJSON), &want) != nil || !reflect.DeepEqual(schema, want["parameters"]) {
			t.Fatalf("schema %s", task.Schema)
		}
		shpContains(t, "feedback", x.inputs[2].Feedback, "Question check: 1 of 1 question(s) are not UI/UX choices left to the user; nothing was asked.")
		shpMatch(t, "usage line", f.Stdout.String(), `shape claude-opus-5-5/xhigh claude-code 5 round\(s\) wall_ms=\d+`)
		noNativeRequests(t, f)
	})
	t.Run("a UI/UX question reaches the user", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(ask), check("ui_ux")}}
		exit, err := f.shapeWith(x, "Add calc.is_even(n).")
		if err != nil || exit != 5 {
			t.Fatalf("exit %d, err %v", exit, err)
		}
		shpContains(t, "stdout", f.Stdout.String(), "1. Which wording?")
	})
	for name, tc := range map[string]struct {
		round func(int, string) (executor.RoundResult, error)
		why   string
	}{
		"not a JSON object": {result([]any{}), "not a JSON object"},
		"an unfinished round": {func(int, string) (executor.RoundResult, error) {
			return executor.RoundResult{Ending: executor.EndNoCall}, nil
		}, "the session ended without a result"},
		"items of another shape": {result(map[string]any{"items": "none"}), "items must be a list"},
	} {
		t.Run(name, func(t *testing.T) {
			parallel(t)
			f := newShpFx(t)
			f.useExternalShape()
			x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(ask), tc.round}}
			exit, err := f.shapeWith(x, "Add calc.is_even(n).")
			shpWantError(t, err, "shape", "question_check_invalid", "", 1)
			shpEq(t, "message", err.Error(), "shape/question_check_invalid: the question check answer is not a valid submit_question_check payload ("+tc.why+"); nothing was asked")
			shpEq(t, "exit", exit, 1)
			if f.exists(".kogen/local/shaping/is-even/questions.json") {
				t.Fatal("nothing is asked")
			}
		})
	}
}

// ---- (evidence: ...) in the Decisions ----

const evHeadline = "Decision evidence: %d problem(s) with what the Decisions cite; nothing was published. A Decision's (evidence: ...) names files this shaping read or directories it listed, tracked at HEAD, or ran `<command>` for a command this shaping ran, separated by \"; \". Where the repository has no evidence or conflicting evidence, mark the Decision (assumption: <the uncertainty you decided under>) instead. Resubmitting this candidate unchanged is refused again."

func TestShapeDecisionEvidenceNative(t *testing.T) {
	parallel(t)
	bad := shpFilesWithDecisions(
		"- Zero is even. (evidence: README.md; features/steps/; `calc.js`) (source: request)",
		"- Negatives are even too. (evidence: nothing/here.rb; lib/) (source: request)",
		"- Odd numbers are not. (evidence: README.md; ; ran `true`) (source: request)",
		"- Floats raise. (evidence: ran `uname -a`; ran `echo \"(x)\"`) (source: request)")
	probe := shpAnswer(shpCall("p1", "probe", map[string]any{"command": `echo "(x)"`}))
	inspect := shpAnswer(shpCall("p2", "list_files", map[string]any{"path": "./features"}), shpCall("p3", "read_file", map[string]any{"path": "./calc.js"}))
	good := shpFilesWithDecisions(
		"- Zero is even. (evidence: README.md; features/steps/; `calc.js`) (source: request)",
		"- Floats raise. (evidence: ran `echo \"(x)\"`) (evidence: `README.md`) (source: request)",
		"- Plain decisions need no evidence. (source: request)")
	f := newShpFx(t, probe, shpSubmit("c1", bad), shpSubmit("c2", bad), inspect, shpSubmit("c3", good), shpAck("c4"))
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	want := strings.Join([]string{
		strings.Replace(evHeadline, "%d", "7", 1),
		"- decision 1: this shaping did not list features/steps/",
		"- decision 1: this shaping did not read calc.js",
		"- decision 2: nothing/here.rb is not a file or directory tracked at HEAD",
		"- decision 2: lib/ is not a file or directory tracked at HEAD",
		"- decision 3: an empty (evidence: ...) item",
		"- decision 3: no probe of this shaping ran `true`",
		"- decision 4: no probe of this shaping ran `uname -a`",
	}, "\n")
	reqs := f.Prov.requests()
	out := shpOutputs(reqs[len(reqs)-2])
	shpEq(t, "first refusal", out["c1"], want)
	shpEq(t, "unchanged resubmission", out["c2"], want+unchangedLine(1))
	refused := shpNamed(f.events(), "submission_refused")
	shpEq(t, "refusals", len(refused), 2)
	shpEq(t, "gate", refused[0]["gate"].(string), "decision_evidence")
	shpEq(t, "problems", len(refused[0]["problems"].([]any)), 7)
	shpEq(t, "published", f.read(".kogen/intents/is-even/intent.md") != "", true)
}

// TestEvidenceProblemsExternalRead: an external session that reported no completed read-only
// inspection call cannot have read a cited file.
func TestEvidenceProblemsExternalRead(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	intent := "## Decisions\n- Zero is even. (evidence: calc.js; features/steps/) (source: request)\n"
	sh := &shaping{repo: f.Repo, ext: &shapeExec{t: t}}
	got, err := sh.evidenceProblems(intent)
	shpEq(t, "err", err, nil)
	shpEq(t, "unread", strings.Join(got, "\n"),
		"- decision 1: this shaping's session read nothing, so calc.js was not read\n- decision 1: this shaping's session read nothing, so features/steps/ was not read")
	sh.inspected = true
	got, err = sh.evidenceProblems(intent)
	shpEq(t, "err", err, nil)
	shpEq(t, "read", len(got), 0)
}

func TestShapeDecisionEvidenceExternal(t *testing.T) {
	parallel(t)
	ran := shpFilesWithDecisions("- Zero is even. (evidence: ran `pytest -q`; calc.js) (source: request)")
	for name, bash := range map[string]bool{"a session that ran no command": false, "a session that ran a command": true} {
		t.Run(name, func(t *testing.T) {
			parallel(t)
			f := newShpFx(t)
			f.useExternalShape()
			x := &shapeExec{t: t, bash: bash, rounds: []func(int, string) (executor.RoundResult, error){result(ran), result(map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}})}}
			exit, err := f.shapeWith(x, "Add calc.is_even(n).")
			if bash {
				if err != nil || exit != 0 {
					t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
				}
				return
			}
			shpWantError(t, err, "intent", "not_ready", "", 1)
			shpEq(t, "feedback", strings.SplitN(x.inputs[1].Feedback, "\n\nCumulative proposal manifest: ", 2)[0],
				strings.Replace(evHeadline, "%d", "1", 1)+"\n- decision 1: this shaping's session ran no command, so `pytest -q` was not run")
		})
	}
}
