package shape

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-executor/executor"
	"kogen-loop/sys"
)

// dsFixture is a synthetic account project: memberships in db/memberships.csv, one export file
// per account, other keys in lib/notes.txt, Kogen's own directory and a binary file.
var dsFixture = map[string]string{
	"lib/account.js":     "const rows = 'db/memberships.csv'; // account_id\n",
	"db/memberships.csv": "account_id,user\n1,ada\n",
	"lib/exports.js":     "// exports/<account_id>.json\nconst a = 1;\nconst b = 2;\nexports.write = id => ({ account_id: id });\n",
	"exports/1.json":     "{\"account_id\":1}\n",
	"exports/2.json":     "{\"account_id\":2}\n",
	"lib/notes.txt":      "my_account_id and account_ids are other keys\n",
	".kogen/notes.txt":   "account_id\n",
	"lib/blob.bin":       "account_id\x00binary\n",
}

func dsCommit(f *shpFx) {
	f.t.Helper()
	for p, c := range dsFixture {
		f.write(p, c)
	}
	f.git("add", "-A")
	f.git("commit", "-qm", "Add the account fixture")
}

const dsRequest = "Purge a cancelled account's data.\n"

func dsIntent(decisions, stores []string) string {
	return "# Purge a cancelled account's data\n\nSlug: is-even\nModel: not applicable — one command removes rows and files\nAcceptance:\n" +
		"- .kogen/intents/is-even/acceptance.feature\n- features/steps/is_even_steps.js\n\n" +
		"The purge removes what the repository keeps for a cancelled account.\n\n" +
		"## Decisions\n" + strings.Join(decisions, "\n") + "\n\n" +
		"## Data stores\n" + strings.Join(stores, "\n") + "\n\n" +
		"## Source comparison\n- addressed: purge a cancelled account's data; source request; contract acceptance.feature.\n\n" +
		"## Tests\n- Purging removes the account's data: the purge scenario (scenario)\n\n## Feasibility\n- none\n"
}

func dsPackage(intent, steps string) map[string]any {
	return map[string]any{"files": []any{
		map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
		map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature},
		map[string]any{"path": "features/steps/is_even_steps.js", "content": steps},
	}}
}

const dsModelDecision = "- `lib/account.js` is the model's code, not a store (source: request)"

var (
	dsPartial = dsIntent(
		[]string{"- Purging removes the memberships rows (source: request)", dsModelDecision},
		[]string{"- key `account_id`", "- `db/memberships.csv`: in scope, observed by `db/memberships.csv`", "- `lib/account.js`: excluded by decision 2"})
	dsCovering = dsIntent(
		[]string{"- Purging removes the memberships rows and the exports (source: request)", dsModelDecision,
			"- `lib/exports.js` writes the exports, not a store (source: request)"},
		[]string{"- key `account_id`", "- `db/memberships.csv`: in scope, observed by `db/memberships.csv`", "- `exports/`: in scope, observed by `exports/`",
			"- `lib/account.js`: excluded by decision 2", "- `lib/exports.js`: excluded by decision 3"})
	dsStepsPartial  = shpSteps + "// Observes db/memberships.csv.\n"
	dsStepsCovering = shpSteps + "// Observes db/memberships.csv and exports/.\n"
)

const dsRefusal = "Data stores: 3 problem(s) with where the data of the declared keys lives; nothing was published. Every file tracked at HEAD that references a key declared under ## Data stores needs an item that covers it: in scope, observed by text that acceptance.feature or a listed step file contains, or excluded by a numbered Decision that names it. Resubmitting this candidate unchanged is refused again.\n" +
	"- exports/1.json references account_id (line 1); no ## Data stores item covers it\n" +
	"- exports/2.json references account_id (line 1); no ## Data stores item covers it\n" +
	"- lib/exports.js references account_id (lines 1, 4); no ## Data stores item covers it"

func dsGates(f *shpFx) []string {
	var gates []string
	for _, e := range shpNamed(f.events(), "submission_refused") {
		gates = append(gates, e["gate"].(string))
	}
	return gates
}

// ---- the scan and the tool's output ----

func TestScanDataStoresReferences(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	dsCommit(f)
	f.write("lib/untracked.js", "account_id\n")
	f.git("add", "lib/untracked.js") // staged, not at HEAD
	refs, err := scanDataStores(f.Repo, []string{"account_id"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range refs {
		got = append(got, r.text())
	}
	want := []string{
		"db/memberships.csv references account_id (line 1)",
		"exports/1.json references account_id (line 1)",
		"exports/2.json references account_id (line 1)",
		"lib/account.js references account_id (line 1)",
		"lib/exports.js references account_id (lines 1, 4)",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("references:\n%q\nwant\n%q", got, want)
	}

	// Two keys: ordered by file, then by key order.
	refs, err = scanDataStores(f.Repo, []string{"user", "account_id"})
	if err != nil {
		t.Fatal(err)
	}
	got = nil
	for _, r := range refs[:2] {
		got = append(got, r.text())
	}
	want = []string{"db/memberships.csv references user (line 1)", "db/memberships.csv references account_id (line 1)"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("two keys:\n%q\nwant\n%q", got, want)
	}
}

func TestDataStoresToolOutput(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	dsCommit(f)
	sh := &shaping{repo: f.Repo}
	const guidance = "Record them in a ## Data stores section of intent.md: a line - key `<key>` for each key, then, for each file above or a directory (a path ending in /) that holds some of them, either - `<path>`: in scope, observed by `<text>`, where acceptance.feature or a step file listed under Acceptance contains <text>, or - `<path>`: excluded by decision <n>, where line <n> of ## Decisions names <path> and keeps its source. Shape refuses a candidate whose section leaves one of these files uncovered."
	const invalid = "data_stores: keys must be a nonempty list of id key spellings without whitespace or backticks; nothing was scanned"
	for _, tc := range []struct {
		name string
		args map[string]any
		want string
	}{
		{"one key", map[string]any{"keys": []any{"account_id"}},
			"Data stores for keys account_id: files tracked at HEAD that reference a key\n" +
				"- db/memberships.csv references account_id (line 1)\n- exports/1.json references account_id (line 1)\n- exports/2.json references account_id (line 1)\n" +
				"- lib/account.js references account_id (line 1)\n- lib/exports.js references account_id (lines 1, 4)\n" + guidance},
		{"duplicates scan once", map[string]any{"keys": []any{"user", "user"}},
			"Data stores for keys user: files tracked at HEAD that reference a key\n- db/memberships.csv references user (line 1)\n" + guidance},
		{"no reference", map[string]any{"keys": []any{"nowhere_id"}},
			"Data stores for keys nowhere_id: files tracked at HEAD that reference a key\n- none\n" + guidance},
		{"two keys", map[string]any{"keys": []any{"user", "nowhere_id"}},
			"Data stores for keys user, nowhere_id: files tracked at HEAD that reference a key\n- db/memberships.csv references user (line 1)\n" + guidance},
		{"missing", map[string]any{}, invalid},
		{"not an array", map[string]any{"keys": "account_id"}, invalid},
		{"empty", map[string]any{"keys": []any{}}, invalid},
		{"not a string", map[string]any{"keys": []any{"a", 1}}, invalid},
		{"empty string", map[string]any{"keys": []any{"a", ""}}, invalid},
		{"whitespace", map[string]any{"keys": []any{"a b"}}, invalid},
		{"backtick", map[string]any{"keys": []any{"a`b"}}, invalid},
	} {
		got, err := sh.dataStores(tc.args)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("%s:\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
}

func TestDataStoresScanGitFailureIsAnError(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.git("rm", "-rq", "--cached", ".")
	f.git("commit", "-qm", "empty")
	sh := &shaping{repo: f.Repo.At(t.TempDir())}
	if _, err := sh.dataStores(map[string]any{"keys": []any{"x"}}); err == nil {
		t.Fatal("a scan outside a repository must be an error")
	}
}

// ---- the tool in the conversation ----

func TestDataStoresToolIsOfferedToNativeShapeOnly(t *testing.T) {
	parallel(t)
	f := newShpFx(t, shpSubmit("c1", shpFiles()))
	f.setReport("PASSED", 0)
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	reqs := f.Prov.requests()
	shpEq(t, "tools", strings.Join(shpToolNames(reqs[0]), ", "), "read_file, view_image, list_files, search_text, probe, documentation, data_stores, propose_files, submit")
	tools, _ := reqs[0]["tools"].([]any)
	var def map[string]any
	for _, d := range tools {
		if m := d.(map[string]any); m["name"] == "data_stores" {
			def = m
		}
	}
	var want map[string]any
	if err := json.Unmarshal([]byte(`{"type":"function","name":"data_stores","description":"Find where an entity's data lives. For a request about an entity's data (for example deleting, exporting or moving an account's data), call this with each spelling of the entity's id key before you write scope and acceptance. It lists every file tracked at HEAD that references a key and how intent.md's ## Data stores section covers each one.","parameters":{"type":"object","properties":{"keys":{"type":"array","items":{"type":"string"}}},"required":["keys"],"additionalProperties":false},"strict":false}`), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(def, want) {
		t.Fatalf("definition %v, want %v", def, want)
	}

	// Refine does not get it.
	r := newRefFx(t, refOpts{literal: true}, responsesWire.readReadme("r1"), responsesWire.refineSubmit("c1", refIntentPath, refTitled))
	if exit, err := r.refine(r.hash0); err != nil || exit != 0 {
		t.Fatalf("refine exit %d: %v", exit, err)
	}
	for i, req := range r.Prov.requests() {
		if strings.Contains(strings.Join(shpToolNames(req), ","), "data_stores") {
			t.Errorf("refine request %d offers data_stores", i+1)
		}
	}
}

func TestDataStoresCallIsAToolCallAndNotAnInspection(t *testing.T) {
	parallel(t)
	w := responsesWire
	call := w.calls(shpCall("ds1", "data_stores", map[string]any{"keys": []any{"account_id"}}))
	f := w.newShape(t, w.prose("Thinking."), call, w.prose("Thinking more."), w.calls(shpCall("c0", "submit", shpFiles())),
		w.readReadme("r"), w.shapeSubmit("c1", shpFiles()), w.shapeAck("c2"))
	dsCommit(f)
	exit, err := f.shape("Add calc.is_even(n).")
	if err != nil || exit != 0 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	reqs := f.Prov.requests()
	// The data_stores call did not count as an inspection: the submission after it was refused.
	shpContains(t, "first submission", shpOutputs(reqs[4])["c0"], "shape/uninspected")
	counts := shpNamed(f.events(), "shaping_turn_counts")
	shpEq(t, "no_tool_turns", counts[0]["no_tool_turns"], any(float64(2)))
	if len(shpOutputs(reqs[2])["ds1"]) == 0 {
		t.Error("no data_stores output was delivered")
	}
}

func TestDataStoresGateRefusesUncoveredFilesAndCoveringPackagePublishes(t *testing.T) {
	parallel(t)
	f := newShpFx(t,
		shpAnswer(shpCall("ds1", "data_stores", map[string]any{"keys": []any{"account_id"}})),
		shpSubmit("c1", dsPackage(dsPartial, dsStepsPartial)),
		shpAnswer(shpCall("c2", "submit", dsPackage(dsPartial, dsStepsPartial))),
		shpAnswer(shpCall("c3", "submit", dsPackage(dsCovering, dsStepsCovering))))
	dsCommit(f)
	f.setReport("PASSED", 0)
	if exit, err := f.shape(dsRequest); err != nil || exit != 0 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	reqs := f.Prov.requests()
	shpEq(t, "first refusal", shpOutputs(reqs[2])["c1"], dsRefusal)
	shpEq(t, "unchanged resubmission", shpOutputs(reqs[3])["c2"], dsRefusal)
	shpEq(t, "gates", strings.Join(dsGates(f), ","), "data_stores,data_stores")
	shpEq(t, "published", f.read(refIntentPath), dsCovering)
	shpEq(t, "acceptance runs", f.runs(), 2)
	for _, e := range shpNamed(f.events(), "submission_refused") {
		if got := e["problems"].([]any); len(got) != 3 {
			t.Errorf("refusal problems %v", got)
		}
	}
}

func TestDataStoresGateEndsAtTheThirdInvalidSubmission(t *testing.T) {
	parallel(t)
	pkg := dsPackage(dsPartial, dsStepsPartial)
	f := newShpFx(t, shpSubmit("c1", pkg), shpAnswer(shpCall("c2", "submit", pkg)), shpAnswer(shpCall("c3", "submit", pkg)))
	dsCommit(f)
	f.setReport("PASSED", 0)
	exit, err := f.shape(dsRequest)
	e, ok := sys.AsError(err)
	if !ok || exit != 1 || e.Class != "intent" || e.Reason != "not_ready" {
		t.Fatalf("exit %d, error %v", exit, err)
	}
	shpEq(t, "lines", strings.Join(e.Lines, "\n"), strings.SplitN(dsRefusal, "\n", 2)[1])
	shpEq(t, "gates", strings.Join(dsGates(f), ","), "data_stores,data_stores,data_stores")
	if f.exists(".kogen/local/readiness/is-even.json") {
		t.Error("a refused candidate recorded readiness")
	}
}

func TestPackageWithoutDataStoresSectionIsNotGated(t *testing.T) {
	parallel(t)
	f := newShpFx(t, shpSubmit("c1", shpFiles()))
	dsCommit(f)
	f.setReport("PASSED", 0)
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	shpEq(t, "gates", len(dsGates(f)), 0)
}

// ---- the section's grammar and its validity ----

func TestDataStoresSectionProblems(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	dsCommit(f)
	f.write(".kogen/intents/is-even/acceptance.feature", shpFeature+"# observes lib/account.js\n")
	f.write("features/steps/is_even_steps.js", shpSteps+"// db/memberships.csv\n")
	sh := &shaping{repo: f.Repo, root: f.Root, slug: shpSlug}
	decisions := []string{"- `lib/account.js` is code (source: request)", "- `exports/` is left to its owner (source: request)"}
	problems := func(stores ...string) []string {
		t.Helper()
		f.write(".kogen/intents/is-even/intent.md", dsIntent(decisions, stores))
		intent, err := project.ReadIntent(f.Root, shpSlug, nil)
		if err != nil {
			t.Fatal(err)
		}
		got, err := sh.dataStoresProblems(intent)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	everything := []string{"- `db/`: in scope, observed by `db/memberships.csv`", "- `lib/`: in scope, observed by `lib/account.js`", "- `exports/`: excluded by decision 2"}
	for _, tc := range []struct {
		name   string
		stores []string
		want   []string
	}{
		{"covered by directories", append([]string{"- key `account_id`"}, everything...), nil},
		{"no key", everything, []string{"- ## Data stores declares no key"}},
		{"malformed lines in order", []string{"- key `a b`", "- key `account_id`", "- `x`: in scope", "- `x`: excluded by decision 0", "- `x`: excluded by decision 02", "- `x`: in scope, observed by ``"},
			[]string{
				"- not a ## Data stores item: - key `a b`", "- not a ## Data stores item: - `x`: in scope", "- not a ## Data stores item: - `x`: excluded by decision 0",
				"- not a ## Data stores item: - `x`: excluded by decision 02", "- not a ## Data stores item: - `x`: in scope, observed by ``",
				"- db/memberships.csv references account_id (line 1); no ## Data stores item covers it",
				"- exports/1.json references account_id (line 1); no ## Data stores item covers it",
				"- exports/2.json references account_id (line 1); no ## Data stores item covers it",
				"- lib/account.js references account_id (line 1); no ## Data stores item covers it",
				"- lib/exports.js references account_id (lines 1, 4); no ## Data stores item covers it"}},
		{"item validity in order", []string{"- key `account_id`", "- `nothing/`: in scope, observed by `lib/account.js`", "- `db/`: in scope, observed by `not in acceptance`",
			"- `exports/`: excluded by decision 3", "- `exports/1.json`: excluded by decision 1", "- `lib/account.js`: excluded by decision 1", "- `lib/exports.js`: in scope, observed by `lib/account.js`"},
			[]string{
				"- nothing/: no file tracked at HEAD has this path",
				"- db/: acceptance.feature and the step files listed under Acceptance do not contain not in acceptance",
				"- exports/: ## Decisions has no line 3",
				"- exports/1.json: decision 1 does not name exports/1.json",
				"- db/memberships.csv references account_id (line 1); no ## Data stores item covers it",
				"- exports/1.json references account_id (line 1); no ## Data stores item covers it",
				"- exports/2.json references account_id (line 1); no ## Data stores item covers it"}},
		{"a file path matches only that file", []string{"- key `account_id`", "- `lib/account.js`: excluded by decision 1", "- `lib/account.j`: excluded by decision 1"},
			[]string{
				"- lib/account.j: no file tracked at HEAD has this path",
				"- db/memberships.csv references account_id (line 1); no ## Data stores item covers it",
				"- exports/1.json references account_id (line 1); no ## Data stores item covers it",
				"- exports/2.json references account_id (line 1); no ## Data stores item covers it",
				"- lib/exports.js references account_id (lines 1, 4); no ## Data stores item covers it"}},
	} {
		if got := problems(tc.stores...); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s:\n%q\nwant\n%q", tc.name, got, tc.want)
		}
	}
}

// ---- the instructions ----

func TestNativeShaperInstructionsCarryNoDataStoresGuidance(t *testing.T) {
	parallel(t)
	ins := ShaperInstructions()
	if strings.Contains(ins, "Data stores") || strings.Contains(ins, "data_stores") {
		t.Fatal("the native shaper's instructions mention data stores")
	}
	sum := sha256.Sum256([]byte(ins))
	if got := hex.EncodeToString(sum[:]); got != "48a1bf625e1b71593f687d64b7a6d85c1c30749180462b97917e83084d1014c4" {
		t.Fatalf("instructions sha256 %s", got)
	}
	if strings.Contains(refineInstructions(), "Data stores") {
		t.Fatal("Refine's instructions mention data stores")
	}
}

// ---- external Shape ----

func TestExternalShapeGetsTheGuidanceAndTheSameRefusal(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape(6)
	dsCommit(f)
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
		result(dsPackage(dsPartial, dsStepsPartial)), result(dsPackage(dsCovering, dsStepsCovering))}}
	f.setReport("PASSED", 0)
	if exit, err := f.shapeWith(x, dsRequest); err != nil || exit != 0 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	shpEq(t, "sessions", len(x.starts), 1)
	stable := x.starts[0].Brief.Stable
	if !strings.HasPrefix(stable, ShaperInstructions()+externalDataStoresGuidance+"\n\n"+externalNote+"\n\n") {
		t.Fatal("the session text is not the instructions, the guidance paragraph and the note")
	}
	if !strings.HasSuffix(externalDataStoresGuidance, ".\n") || strings.Count(externalDataStoresGuidance, "\n") != 1 {
		t.Fatal("the guidance paragraph is one line ending in one newline")
	}
	if strings.Contains(ShaperInstructions(), "Data stores") {
		t.Fatal("the guidance is not part of the base instructions")
	}
	shpEq(t, "feedback", strings.SplitN(x.inputs[1].Feedback, "\n\nCumulative proposal manifest: ", 2)[0], dsRefusal)
	shpContains(t, "feedback", x.inputs[1].Feedback, "\n\nCumulative proposal manifest: ")
	shpEq(t, "gates", strings.Join(dsGates(f), ","), "data_stores")
	shpEq(t, "published", f.read(refIntentPath), dsCovering)
}
