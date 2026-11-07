package shape

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-executor/executor"
)

// dlAsk is a questions submission with decisions.
func dlAsk(question string, decisions ...any) map[string]any {
	args := map[string]any{"questions": []any{shpQuestion(question, "keep", "shorter")}}
	if len(decisions) > 0 {
		args["decisions"] = decisions
	}
	return args
}

// dlCheck is the question check answer for one question: its class, why and decision.
func dlCheck(id, class, decision string) shpStep {
	return shpAnswer(shpCall(id, "submit_question_check", map[string]any{"items": []any{
		map[string]any{"number": 1, "class": class, "why": "a " + class + " question", "decision": decision},
	}}))
}

func dlRead(path string) map[string]any {
	return shpCall("r-"+path, "read_file", map[string]any{"path": path})
}

func dlLedger(t *testing.T, f *shpFx) *decisionLedger {
	t.Helper()
	l, err := readLedger(filepath.Join(f.Root, ".kogen/local/shaping", shpSlug))
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func dlSha(t *testing.T, f *shpFx, rel string) string {
	t.Helper()
	return ContentHash([]byte(f.read(rel)))
}

func TestDecisionLedgerText(t *testing.T) {
	parallel(t)
	d1 := recordedDecision{ID: "D1", Round: 1, Decision: "Keep safe diagnostics", Constraints: []string{"no record bytes", "no paths"}, SupersededBy: "D3",
		Evidence: []receipt{
			{Item: "a.md", Kind: receiptRead, Round: 1, SHA256: "old", Text: "1\tone\n\n"},
			{Item: "ran `ls`", Kind: receiptProbe, Round: 1, Text: "exit status: 0\n"},
		}}
	d2 := recordedDecision{ID: "D2", Round: 2, Decision: "Reads stay", Uncertainty: "no test to cite", Evidence: []receipt{}}
	d3 := recordedDecision{ID: "D3", Round: 3, Decision: "Keep safe diagnostics and size", Constraints: []string{"no record bytes"}, Revises: "D1", Why: "policy changed",
		Evidence: []receipt{{Item: "a.md", Kind: receiptCheckout, Round: 3, SHA256: "new", Text: "1\ttwo"}}}
	l := &decisionLedger{Decisions: []recordedDecision{d1, d2, d3}}
	current := func(string) string { return "new" }

	shpEq(t, "line d1", d1.intentLine(), "- Keep safe diagnostics (decision: D1, round 1) (constraint: no record bytes) (constraint: no paths) (evidence: a.md; ran `ls`) (source: request)")
	shpEq(t, "line d2", d2.intentLine(), "- Reads stay (decision: D2, round 2) (assumption: no test to cite) (source: request)")
	shpEq(t, "line d3", d3.intentLine(), "- Keep safe diagnostics and size (decision: D3, round 3) (constraint: no record bytes) (evidence: a.md) (revises: D1 because policy changed) (source: request)")

	shpEq(t, "check input", l.checkInputText(current), "\n\n# Recorded decisions\n\n"+
		"D2 (round 2): Reads stay\n   uncertainty: no test to cite\n\n"+
		"D3 (round 3): Keep safe diagnostics and size\n   constraint: no record bytes\n   revises: D1 because policy changed\n   evidence: a.md (read from the checkout in round 3)\n\n"+
		"D3 cites a.md (read from the checkout in round 3):\n1\ttwo")
	shpEq(t, "no standing decision", (&decisionLedger{Decisions: []recordedDecision{{ID: "D1", SupersededBy: "D2"}}}).checkInputText(current), "")
	shpEq(t, "nil ledger", (*decisionLedger)(nil).checkInputText(current), "")

	shpEq(t, "context", l.contextText(current), contextDecisionsHeading+
		"D1 (round 1, revised by D3): Keep safe diagnostics\n   constraint: no record bytes\n   constraint: no paths\n   evidence: a.md (read by this shaping in round 1; changed since)\n   evidence: ran `ls` (probe output in round 1)\n\n"+
		"D2 (round 2): Reads stay\n   uncertainty: no test to cite\n   Intent line: "+d2.intentLine()+"\n\n"+
		"D3 (round 3): Keep safe diagnostics and size\n   constraint: no record bytes\n   revises: D1 because policy changed\n   evidence: a.md (read from the checkout in round 3)\n   Intent line: "+d3.intentLine()+"\n")
	shpEq(t, "empty context", (&decisionLedger{}).contextText(current), "")
}

func TestShapeDecisionSubmissionValidation(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	cfg, err := project.LoadConfig(f.Root)
	if err != nil {
		t.Fatal(err)
	}
	sh := &shaping{root: f.Root, slug: shpSlug, cfg: cfg}
	ask := func(decisions any) map[string]any {
		return map[string]any{"questions": []any{shpQuestion("Which wording?")}, "decisions": decisions}
	}
	for name, c := range map[string]struct {
		args map[string]any
		want string
	}{
		"beside blocked": {map[string]any{"blocked": map[string]any{"paths": []any{"x"}, "reason": "r"}, "decisions": []any{map[string]any{"decision": "d"}}}, "decisions go with questions or files, not with a blocked report"},
		"not a list":     {ask("d"), decisionsMalformed},
		"not an object":  {ask([]any{"d"}), decisionsMalformed},
		"blank decision": {ask([]any{map[string]any{"decision": "  "}}), decisionsMalformed},
		"no decision":    {ask([]any{map[string]any{"uncertainty": "u"}}), decisionsMalformed},
		"evidence":       {ask([]any{map[string]any{"decision": "d", "evidence": []any{1}}}), decisionsMalformed},
		"constraints":    {ask([]any{map[string]any{"decision": "d", "constraints": "c"}}), decisionsMalformed},
		"uncertainty":    {ask([]any{map[string]any{"decision": "d", "uncertainty": 1}}), decisionsMalformed},
		"revises":        {ask([]any{map[string]any{"decision": "d", "revises": []any{}}}), decisionsMalformed},
		"why":            {ask([]any{map[string]any{"decision": "d", "why": false}}), decisionsMalformed},
	} {
		sub, problems, err := sh.validate(c.args)
		if err != nil || len(problems) == 0 || problems[len(problems)-1] != c.want || sub.decisions != nil || sub.questions != nil {
			t.Errorf("%s: %+v, %q, %v", name, sub, problems, err)
		}
	}
	for name, decisions := range map[string]any{"empty list": []any{}, "valid": []any{map[string]any{"decision": "d", "uncertainty": "u"}}} {
		sub, problems, err := sh.validate(ask(decisions))
		if err != nil || problems != nil || len(sub.questions) != 1 || len(sub.decisions) != len(decisions.([]any)) {
			t.Errorf("%s: %+v, %q, %v", name, sub, problems, err)
		}
	}
}

// TestShapeDecisionRecordGate: the problem lines of a submission's decisions, the ledger a good
// one writes and the event that records it.
func TestShapeDecisionRecordGate(t *testing.T) {
	parallel(t)
	evidence := func(items ...string) map[string]any {
		list := []any{}
		for _, it := range items {
			list = append(list, it)
		}
		return map[string]any{"decision": "A", "evidence": list, "constraints": []any{"c1"}}
	}
	bad := dlAsk("Which wording?",
		evidence("README.md", "calc.js", "features/steps/", "nothing.rb", " ", "ran `false`", "ran `true`", "ran `never`"),
		map[string]any{"decision": "B"},
		map[string]any{"decision": "C", "uncertainty": "u", "revises": "D9"},
		map[string]any{"decision": "D", "uncertainty": "u", "revises": "D1", "why": "w"})
	good := dlAsk("Which wording?",
		evidence("calc.js", "ran `true`"),
		map[string]any{"decision": "B", "uncertainty": "no test"})
	f := newShpFx(t,
		shpAnswer(dlRead("calc.js"), shpCall("p1", "probe", map[string]any{"command": "true"}), shpCall("p2", "probe", map[string]any{"command": "false"})),
		shpSubmit("s1", bad), shpSubmit("s2", good), shpUI(1))
	exit, err := f.shape("Add calc.is_even(n).")
	if err != nil || exit != 5 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	reqs := shpShaperRequests(f.Prov.requests())
	shpEq(t, "feedback", shpOutputs(reqs[len(reqs)-1])["s1"], strings.Join([]string{
		strings.Replace(decisionRecordHeadline, "%d", "10", 1),
		"- sourced decision 1: this shaping did not read README.md",
		"- sourced decision 1: features/steps/ is a directory; cite the files this shaping read",
		"- sourced decision 1: nothing.rb is not a file tracked at HEAD",
		"- sourced decision 1: an empty evidence item",
		"- sourced decision 1: the probe of `false` failed (exit status: 1), so it is no evidence",
		"- sourced decision 1: no probe of this shaping ran `never`",
		"- sourced decision 2: no evidence and no uncertainty; cite what this shaping read or ran, or state the uncertainty you decided under",
		"- sourced decision 3: revises D9 without why",
		"- sourced decision 3: D9 is not a recorded decision that stands",
		"- sourced decision 4: D1 is not a recorded decision that stands",
	}, "\n"))
	refused := shpNamed(f.events(), "submission_refused")
	shpEq(t, "refusals", len(refused), 1)
	shpEq(t, "gate", refused[0]["gate"].(string), "decision_record")

	// The refusal asked and recorded nothing; the good submission recorded both decisions.
	ledger := dlLedger(t, f)
	shpEq(t, "decisions", len(ledger.Decisions), 2)
	d1 := ledger.Decisions[0]
	shpEq(t, "id", d1.ID, "D1")
	shpEq(t, "round", d1.Round, 1)
	shpEq(t, "constraints", strings.Join(d1.Constraints, ","), "c1")
	shpEq(t, "receipts", len(d1.Evidence), 2)
	shpEq(t, "read receipt", d1.Evidence[0], receipt{Item: "calc.js", Kind: receiptRead, Round: 1, SHA256: dlSha(t, f, "calc.js"), Text: "1\tfunction add(a, b) { return a + b }"})
	shpEq(t, "probe receipt kind", d1.Evidence[1].Kind, receiptProbe)
	shpContains(t, "probe receipt text", d1.Evidence[1].Text, "exit status: 0\ntimed_out: false\n")
	shpEq(t, "no evidence", len(ledger.Decisions[1].Evidence), 0)
	shpEq(t, "uncertainty", ledger.Decisions[1].Uncertainty, "no test")
	recorded := shpNamed(f.events(), "decisions_recorded")
	shpEq(t, "events", len(recorded), 1)
	shpEq(t, "recorded", strings.Join(anyStrings(recorded[0]["recorded"]), ","), "D1,D2")
	shpEq(t, "kept", len(anyStrings(recorded[0]["kept"])), 0)
	st, err := os.Stat(filepath.Join(f.Root, ".kogen/local/shaping/is-even/decisions.json"))
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("ledger %v, %v", st, err)
	}
}

func anyStrings(v any) []string {
	var out []string
	list, _ := v.([]any)
	for _, e := range list {
		out = append(out, e.(string))
	}
	return out
}

// TestShapeDecisionLedger: round 1 records, round 2 keeps an equal decision, round 3 refuses a
// revision on unread evidence and records one on evidence it read, round 4 publishes an Intent
// that carries what stands, publication keeps the ledger and a new shaping removes it.
func TestShapeDecisionLedger(t *testing.T) {
	parallel(t)
	d1 := map[string]any{"decision": "Safe diagnostics", "evidence": []any{"calc.js"}, "constraints": []any{"no bytes"}}
	d2 := map[string]any{"decision": "Reads stay", "uncertainty": "no test"}
	unread := map[string]any{"decision": "Safe diagnostics and size", "evidence": []any{"README.md"}, "constraints": []any{"no bytes"}, "revises": "D1", "why": "size needed"}
	revision := map[string]any{"decision": "Safe diagnostics and size", "evidence": []any{"calc.js"}, "constraints": []any{"no bytes"}, "revises": "D1", "why": "size needed"}
	line3 := "- Safe diagnostics and size (decision: D3, round 3) (constraint: no bytes) (evidence: calc.js) (revises: D1 because size needed) (source: request)"
	line2 := "- Reads stay (decision: D2, round 1) (assumption: no test) (source: request)"
	f := newShpFx(t,
		shpAnswer(dlRead("calc.js")), shpSubmit("a1", dlAsk("Which wording?", d1, d2)), shpUI(1),
		shpSubmit("b1", dlAsk("Which icon?", d1, d2)), shpUI(1),
		shpSubmit("c1", dlAsk("Which engine?", unread)),
		shpAnswer(dlRead("calc.js")), shpSubmit("c2", dlAsk("Which engine?", revision)), shpUI(1),
		shpSubmit("d1", shpFilesWithDecisions(line3, line2)), shpAck("d2"),
		shpSubmit("e1", dlAsk("Which wording?")), shpUI(1))
	ledgerPath := ".kogen/local/shaping/is-even/decisions.json"
	for i, input := range []string{"Add calc.is_even(n).", "1: a", "1: a"} {
		if exit, err := f.shape(input); err != nil || exit != 5 {
			t.Fatalf("run %d: exit %d, err %v\n%s", i+1, exit, err, f.Stderr)
		}
	}
	recorded := shpNamed(f.events(), "decisions_recorded")
	shpEq(t, "events", len(recorded), 3)
	shpEq(t, "round 2 records nothing new", len(anyStrings(recorded[1]["recorded"])), 0)
	shpEq(t, "round 2 keeps both", strings.Join(anyStrings(recorded[1]["kept"]), ","), "D1,D2")
	shpEq(t, "round 3 records D3", strings.Join(anyStrings(recorded[2]["recorded"]), ","), "D3")
	var feedback string
	for _, r := range shpShaperRequests(f.Prov.requests()) {
		if out, ok := shpOutputs(r)["c1"]; ok {
			feedback = out
		}
	}
	shpContains(t, "refused revision", feedback, "- sourced decision 1: this shaping did not read README.md")
	shpContains(t, "refused revision", feedback, "- sourced decision 1: revises D1 without evidence this round read or ran, so D1 stands")
	ledger := dlLedger(t, f)
	shpEq(t, "decisions", len(ledger.Decisions), 3)
	shpEq(t, "D1 superseded", ledger.Decisions[0].SupersededBy, "D3")
	shpEq(t, "D1 stays on record", ledger.Decisions[0].Decision, "Safe diagnostics")
	shpEq(t, "D2 stands", ledger.Decisions[1].stands(), true)
	shpEq(t, "D3 round", ledger.Decisions[2].Round, 3)
	shpEq(t, "D3 revises", ledger.Decisions[2].Revises, "D1")

	if exit, err := f.shape("1: a"); err != nil || exit != 0 {
		t.Fatalf("round 4: exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	if !f.exists(".kogen/intents/is-even/intent.md") || !f.exists(ledgerPath) {
		t.Fatal("the Intent is published and the ledger is kept")
	}
	carried := shpNamed(f.events(), "carried_evidence")
	if len(carried) == 0 {
		t.Fatal("no carried_evidence event")
	}
	items := carried[0]["items"].([]any)
	shpEq(t, "carried items", len(items), 1)
	item := items[0].(map[string]any)
	shpEq(t, "carried decision", item["decision"].(float64), float64(1))
	shpEq(t, "carried item", item["item"].(string), "calc.js")
	shpEq(t, "carried kind", item["kind"].(string), "read")
	shpEq(t, "carried round", item["round"].(float64), float64(3))

	if err := os.RemoveAll(filepath.Join(f.Root, ".kogen/intents/is-even")); err != nil {
		t.Fatal(err)
	}
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 5 {
		t.Fatalf("new shaping: exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	if f.exists(ledgerPath) {
		t.Fatal("a new shaping removes the ledger")
	}
}

// TestShapeDecisionLedgerUnreadable: a continuation whose ledger cannot be decoded fails with
// that error.
func TestShapeDecisionLedgerUnreadable(t *testing.T) {
	parallel(t)
	f := newShpFx(t, shpSubmit("a1", dlAsk("Which wording?")), shpUI(1))
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 5 {
		t.Fatalf("exit %d, err %v", exit, err)
	}
	f.write(".kogen/local/shaping/is-even/decisions.json", "{")
	_, err := f.shape("1: a")
	if err == nil || !strings.Contains(err.Error(), "decisions.json") {
		t.Fatalf("err = %v", err)
	}
}

// TestShapeCarriedDecisionsGate: an Intent must carry the line of each decision that stands.
func TestShapeCarriedDecisionsGate(t *testing.T) {
	parallel(t)
	d1 := map[string]any{"decision": "Safe diagnostics", "evidence": []any{"calc.js"}, "constraints": []any{"no bytes"}}
	line := "- Safe diagnostics (decision: D1, round 1) (constraint: no bytes) (evidence: calc.js) (source: request)"
	f := newShpFx(t,
		shpAnswer(dlRead("calc.js")), shpSubmit("a1", dlAsk("Which wording?", d1)), shpUI(1),
		shpSubmit("b1", shpFilesWithDecisions("- Safe diagnostics (source: request)")),
		shpSubmit("b2", shpFilesWithDecisions(line)), shpAck("b3"))
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 5 {
		t.Fatalf("exit %d, err %v", exit, err)
	}
	if exit, err := f.shape("1: a"); err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	var feedback string
	for _, r := range shpShaperRequests(f.Prov.requests()) {
		if out, ok := shpOutputs(r)["b1"]; ok {
			feedback = out
		}
	}
	shpEq(t, "feedback", feedback, strings.Join([]string{
		strings.Replace(carriedDecisionsHeadline, "%d", "1", 1),
		"- D1 (round 1) is not in Decisions; keep it as: " + line,
	}, "\n"))
	refused := shpNamed(f.events(), "submission_refused")
	shpEq(t, "gate", refused[len(refused)-1]["gate"].(string), "carried_decisions")
}

// TestShapeDecisionRecordExternal: the external shaper's receipts are checkout texts, its commands
// are no evidence and an unchanged file is carried to the next round.
func TestShapeDecisionRecordExternal(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	ran := map[string]any{"decision": "Safe", "evidence": []any{"ran `ls`", "calc.js"}}
	ok := map[string]any{"decision": "Safe", "evidence": []any{"calc.js"}}
	x := &shapeExec{t: t, bash: true, rounds: []func(int, string) (executor.RoundResult, error){
		result(dlAsk("Which wording?", ran)), result(dlAsk("Which wording?", ok)), result(map[string]any{"items": []any{map[string]any{"number": 1, "class": "ui_ux", "why": "w", "decision": ""}}}),
	}}
	exit, err := f.shapeWith(x, "Add calc.is_even(n).")
	if err != nil || exit != 5 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	shpEq(t, "feedback", strings.Contains(x.inputs[1].Feedback, "- sourced decision 1: Kogen keeps no output of the external session's commands, so `ls` is no evidence; cite a file this session read"), true)
	ledger := dlLedger(t, f)
	shpEq(t, "decisions", len(ledger.Decisions), 1)
	shpEq(t, "receipt", ledger.Decisions[0].Evidence[0], receipt{Item: "calc.js", Kind: receiptCheckout, Round: 1, SHA256: dlSha(t, f, "calc.js"), Text: "1\tfunction add(a, b) { return a + b }"})
}

// TestEvidenceCheckCarriedReceipts: the evidence gate accepts a receipt of an earlier round for an
// unchanged file or probe, and names a changed file.
func TestEvidenceCheckCarriedReceipts(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	digest := dlSha(t, f, "calc.js")
	ledger := &decisionLedger{Decisions: []recordedDecision{{ID: "D1", Round: 2, Evidence: []receipt{
		{Item: "calc.js", Kind: receiptRead, Round: 2, SHA256: digest, Text: "t"},
		{Item: "README.md", Kind: receiptRead, Round: 1, SHA256: "other", Text: "t"},
		{Item: "ran `ls`", Kind: receiptProbe, Round: 1, Text: "t"},
	}}}}
	intent := "## Decisions\n- A (evidence: calc.js; README.md; ran `ls`; ran `pwd`) (source: request)\n"
	sh := &shaping{root: f.Root, repo: f.Repo, ledger: ledger}
	problems, carried, err := sh.evidenceCheck(intent)
	if err != nil {
		t.Fatal(err)
	}
	shpEq(t, "problems", strings.Join(problems, "\n"), "- decision 1: README.md changed since round 1 read it; read it again\n- decision 1: no probe of this shaping ran `pwd`")
	shpEq(t, "carried", len(carried), 2)
	shpEq(t, "carried file", carried[0], carriedItem{1, "calc.js", ledger.Decisions[0].Evidence[0]})
	shpEq(t, "carried probe", carried[1].item, "ran `ls`")
	sh.readPaths = map[string]bool{"README.md": true}
	problems, _, _ = sh.evidenceCheck(intent)
	shpEq(t, "read again", strings.Join(problems, "\n"), "- decision 1: no probe of this shaping ran `pwd`")
}

// TestShapeDecidedQuestions: a question a recorded decision settles is refused with the decision's
// Intent line and counted as decided, an unknown decision id makes it technical, and the question
// check is shown the decisions with the text they cite.
func TestShapeDecidedQuestions(t *testing.T) {
	parallel(t)
	d1 := map[string]any{"decision": "Safe diagnostics", "evidence": []any{"calc.js"}, "constraints": []any{"no bytes"}}
	f := newShpFx(t,
		shpAnswer(dlRead("calc.js")), shpSubmit("a1", dlAsk("Which wording?", d1)), shpUI(1),
		shpSubmit("b1", dlAsk("Attach bytes?")), dlCheck("c1", "decided", "D1"),
		shpSubmit("b2", dlAsk("Attach bytes or not?")), dlCheck("c2", "decided", "D7"),
		shpSubmit("b3", dlAsk("Which icon?")), dlCheck("c3", "ui_ux", ""))
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 5 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	if exit, err := f.shape("1: a"); err != nil || exit != 5 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	var feedback1, feedback2 string
	for _, r := range shpShaperRequests(f.Prov.requests()) {
		if out, ok := shpOutputs(r)["b1"]; ok {
			feedback1 = out
		}
		if out, ok := shpOutputs(r)["b2"]; ok {
			feedback2 = out
		}
	}
	shpContains(t, "decided feedback", feedback1, "\n- question 1 (decided): Attach bytes? | why: a decided question | D1 (round 1) settles it: Safe diagnostics; do not ask it again. "+
		"Complete the choice from D1, its constraints and its evidence, and keep it in the Intent's Decisions as: - Safe diagnostics (decision: D1, round 1) (constraint: no bytes) (evidence: calc.js) (source: request)")
	shpContains(t, "unknown decision", feedback2, "\n- question 1 (technical): Attach bytes or not? | why: a decided question | decide it from what the repository and the request establish")
	checks := qcChecks(f.Prov.requests())
	shpEq(t, "question checks", len(checks), 4)
	shpContains(t, "first check", qcText(checks[0]), "# Earlier questions and answers\n\nnone\n\n# Recorded decisions\n\nD1 (round 1): Safe diagnostics\n")
	shpContains(t, "second check", qcText(checks[1]), "# Earlier questions and answers\n\nRound 1:\n1. Which wording?\n   a) keep\n   b) shorter\n   answer: 1: a\n\n# Recorded decisions\n\n"+
		"D1 (round 1): Safe diagnostics\n   constraint: no bytes\n   evidence: calc.js (read by this shaping in round 1)\n\n"+
		"D1 cites calc.js (read by this shaping in round 1):\n1\tfunction add(a, b) { return a + b }\n\n# Questions\n\n1. Attach bytes?")
	var classes, decisions []any
	for _, ev := range shpNamed(f.events(), "question_check") {
		item := ev["items"].([]any)[0].(map[string]any)
		classes, decisions = append(classes, item["class"]), append(decisions, item["decision"])
	}
	shpEq(t, "classes", strings.Join(anyStrings(classes), ","), "ui_ux,decided,technical,ui_ux")
	shpEq(t, "decision of the decided question", decisions[1], any("D1"))
	shpEq(t, "no decision on others", decisions[2], any(nil))
}

// TestShapeDecisionCarriedItemAsCited: a carried receipt keeps the earlier round's kind, round,
// digest and text under the item as cited now, so a later round keeps the unchanged decision
// instead of recording it again; a probe receipt is carried too.
func TestShapeDecisionCarriedItemAsCited(t *testing.T) {
	parallel(t)
	d1 := map[string]any{"decision": "Safe diagnostics", "evidence": []any{"./calc.js", "ran `true`"}}
	d2 := map[string]any{"decision": "Reads stay", "evidence": []any{"calc.js", "ran `true`"}}
	f := newShpFx(t,
		shpAnswer(dlRead("calc.js"), shpCall("p1", "probe", map[string]any{"command": "true"})),
		shpSubmit("a1", dlAsk("Which wording?", d1)), shpUI(1),
		shpSubmit("b1", dlAsk("Which icon?", d1, d2)), shpUI(1),
		shpSubmit("c1", dlAsk("Which colour?", d1, d2)), shpUI(1))
	for i, input := range []string{"Add calc.is_even(n).", "1: a", "1: a"} {
		if exit, err := f.shape(input); err != nil || exit != 5 {
			t.Fatalf("run %d: exit %d, err %v\n%s", i+1, exit, err, f.Stderr)
		}
	}
	ledger := dlLedger(t, f)
	shpEq(t, "decisions", len(ledger.Decisions), 2)
	d := ledger.Decisions[1]
	shpEq(t, "carried file item", d.Evidence[0].Item, "calc.js")
	shpEq(t, "carried file round", d.Evidence[0].Round, 1)
	shpEq(t, "carried file text", d.Evidence[0].Text, ledger.Decisions[0].Evidence[0].Text)
	shpEq(t, "carried probe", d.Evidence[1], ledger.Decisions[0].Evidence[1])
	shpEq(t, "intent line", d.intentLine(), "- Reads stay (decision: D2, round 2) (evidence: calc.js; ran `true`) (source: request)")
	recorded := shpNamed(f.events(), "decisions_recorded")
	shpEq(t, "events", len(recorded), 3)
	shpEq(t, "round 3 records nothing", len(anyStrings(recorded[2]["recorded"])), 0)
	shpEq(t, "round 3 keeps both", strings.Join(anyStrings(recorded[2]["kept"]), ","), "D1,D2")
	st, err := os.Stat(filepath.Join(f.Root, ".kogen/local/shaping/is-even"))
	if err != nil || st.Mode().Perm() != 0o700 {
		t.Fatalf("ledger directory %v, %v", st, err)
	}
}

// A files submission records and carries its new decision in the same round.
func TestShapeFilesRecordAndCarryDecisions(t *testing.T) {
	d1 := map[string]any{"decision": "First", "evidence": []any{"calc.js"}}
	d2 := map[string]any{"decision": "Second", "evidence": []any{"calc.js"}, "constraints": []any{"one", "two"}}
	first := "- First (decision: D1, round 1) (evidence: calc.js) (source: request)"
	second := "- Second (decision: D2, round 2) (constraint: one) (constraint: two) (evidence: calc.js) (source: request)"
	missing := shpFilesWithDecisions(first)
	missing["decisions"] = []any{d2}
	dropped := shpFilesWithDecisions(first + "\n" + strings.Replace(second, " (constraint: two)", "", 1))
	dropped["decisions"] = []any{d2}
	good := shpFilesWithDecisions(first + "\n" + second)
	good["decisions"] = []any{d2}
	f := newShpFx(t, shpAnswer(dlRead("calc.js")), shpSubmit("a", dlAsk("Which wording?", d1)), shpUI(1), shpSubmit("missing", missing), shpSubmit("dropped", dropped), shpSubmit("good", good), shpAck("ack"))
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 5 {
		t.Fatalf("first: %d %v", exit, err)
	}
	if exit, err := f.shape("1: a"); err != nil || exit != 0 {
		t.Fatalf("second: %d %v\n%s", exit, err, f.Stderr)
	}
	var feedback string
	for _, r := range shpShaperRequests(f.Prov.requests()) {
		if out, ok := shpOutputs(r)["missing"]; ok {
			feedback = out
		}
	}
	if !strings.Contains(feedback, second) {
		t.Errorf("missing feedback: %s", feedback)
	}
	published := f.read(".kogen/intents/is-even/intent.md")
	if !strings.Contains(published, second) {
		t.Errorf("published draft missing carried decision: %s", published)
	}
	ledger := dlLedger(t, f)
	shpEq(t, "recorded", len(ledger.Decisions), 2)
	shpEq(t, "second line", ledger.Decisions[1].intentLine(), second)
}

// Scripted-provider plumbing only: this does not establish live-model compliance.
func TestShapeSyntheticWebRequestFirstPackage(t *testing.T) {
	line := "- Use existing routing (decision: D1, round 1) (constraint: preserve navigation) (evidence: calc.js) (source: request)"
	pkg := shpFilesWithDecisions(line)
	pkg["decisions"] = []any{map[string]any{"decision": "Use existing routing", "constraints": []any{"preserve navigation"}, "evidence": []any{"calc.js"}}}
	files := pkg["files"].([]any)
	files = append(files, map[string]any{"path": ".kogen/intents/is-even/controls.json", "content": `{"correct":{"files":{"web/app.js":"correct","web/view.js":"correct"}},"wrong":[{"name":"navigation wrong","scenario":"Even numbers","step":"Then is-even: it is even","files":{"web/view.js":"wrong"}}]}`})
	pkg["files"] = files
	f := newShpFx(t, shpAnswer(dlRead("calc.js")), shpSubmit("package", pkg), shpAck("ack"))
	exit, err := f.shape("Update the web application across its routing and view files, preserving navigation and showing the requested page.")
	if err != nil || exit != 0 {
		t.Fatalf("exit %d: %v\n%s", exit, err, f.Stderr)
	}
	if len(files) != 4 || files[3].(map[string]any)["path"] != ".kogen/intents/is-even/controls.json" {
		t.Fatal("first package lacks controls")
	}
	scenarios, err := project.ParseGherkin(shpFeature)
	if err != nil {
		t.Fatal(err)
	}
	controls := files[3].(map[string]any)["content"].(string)
	if _, problems := parseControls([]byte(controls), scenarios, strings.Split(shpFeature, "\n"), controlsRules{}); len(problems) != 0 {
		t.Fatalf("first package controls: %v", problems)
	}
	for _, ev := range shpNamed(f.events(), "submission_refused") {
		gate, _ := ev["gate"].(string)
		if gate == "carried_decisions" || gate == "decision_record" || strings.Contains(fmt.Sprint(ev["problems"]), "keyword included:") {
			t.Fatalf("unexpected refusal: %v", ev)
		}
	}
}

func TestShapeDecisionRejectsPackageAddedEvidence(t *testing.T) {
	bad := shpFiles()
	bad["decisions"] = []any{map[string]any{"decision": "Use new step", "evidence": []any{"features/steps/is-even__steps.js"}}}
	f := newShpFx(t, shpSubmit("bad", bad), shpSubmit("good", shpFiles()), shpAck("ack"))
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("exit %d: %v\n%s", exit, err, f.Stderr)
	}
	var feedback string
	for _, r := range shpShaperRequests(f.Prov.requests()) {
		if out, ok := shpOutputs(r)["bad"]; ok {
			feedback = out
		}
	}
	if !strings.Contains(feedback, "features/steps/is-even__steps.js is not a file tracked at HEAD") {
		t.Fatalf("feedback: %s", feedback)
	}
}
