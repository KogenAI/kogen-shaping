'use strict';
// The Go source of the shape-repair-before-terminal-submit acceptance harness. The steps compile
// it into the shape package's tests through a Go overlay as shape/srb_acceptance_test.go; that
// path never exists in the repository. The source holds no backquote and no dollar-brace, so this
// raw template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"kogen-loop/sys"
)

// The acceptance harness of shape-repair-before-terminal-submit: one scripted native Shape
// journey per run against the loopback provider (no model), in a synthetic repository. SRB_SCRIPT
// names the journey and SRB_WIRE the route (responses or anthropic). The run prints one SRB line
// with what a user and the provider observed.

const srbRecovery = "Recovery before submitting again:"

const srbRun = "NODE_PATH=\"$KOGEN_CUCUMBER\" \"$KOGEN_NODE\" \"$KOGEN_CUCUMBER/@cucumber/cucumber/bin/cucumber.js\" --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\""

var srbConfig = "version: 1\ncommands:\n  setup: []\n  checks: [{name: unit, run: 'true'}]\n  acceptance:\n    run: '" + strings.ReplaceAll(srbRun, "'", "''") + "'\n    steps_dir: features/steps\n"

const (
	srbIntentPath  = ".kogen/intents/is-even/intent.md"
	srbFeaturePath = ".kogen/intents/is-even/acceptance.feature"
	srbStepsPath   = "features/steps/is-even__steps.js"
)

func srbIntent(title, prose string) string {
	return "# " + title + "\n\nSlug: is-even\nModel: not applicable — a function without state\nAcceptance:\n- " + srbFeaturePath + "\n- " + srbStepsPath + "\n\n" + prose +
		"\n\n## Decisions\n- The output format follows the request. (source: request)\n\n## Source comparison\n- The request asks for this behaviour; the feature covers it.\n\n## Tests\n- The requested behaviour: the acceptance scenarios (scenario)\n\n## Feasibility\n- none\n"
}

func srbFile(path, content string) any { return map[string]any{"path": path, "content": content} }

func srbFiles(files ...any) map[string]any { return map[string]any{"files": files} }

// ---- the unprepared database journeys ----

const srbAccounts = "const fs = require('node:fs');\n" +
	"if (!fs.existsSync('db/accounts.json')) { console.error('missing accounts database'); process.exit(1); }\n" +
	"const args = process.argv.slice(2);\n" +
	"if (args.length > 0) { console.error('unknown option ' + args[0]); process.exit(2); }\n" +
	"console.log(JSON.parse(fs.readFileSync('db/accounts.json', 'utf8')).accounts.join('\\n'));\n"

const srbAccountsDB = "{\"accounts\":[\"alice\",\"bob\"],\"active\":[\"alice\"]}\n"

const srbAccountsFeature = "Feature: Active accounts\n  Scenario: List active accounts\n    When is-even: I list the active accounts\n    Then is-even: the output is \"alice\"\n"

var srbAccountsIntent = srbIntent("List only the active accounts", "accounts.js --active lists only the active accounts.")

// srbUnpreparedSteps read the database the checkout has prepared and the fresh copy lacks.
const srbUnpreparedSteps = "const { When, Then } = require('@cucumber/cucumber');\n" +
	"const { spawnSync } = require('node:child_process');\n" +
	"const assert = require('node:assert');\n" +
	"When('is-even: I list the active accounts', function () {\n" +
	"  const r = spawnSync(process.execPath, ['accounts.js', '--active'], { encoding: 'utf8' });\n" +
	"  this.out = r.stdout + r.stderr;\n" +
	"});\n" +
	"Then('is-even: the output is {string}', function (want) {\n" +
	"  assert.strictEqual(this.out, want + '\\n');\n" +
	"});\n"

// srbPreparedSteps prepare the database in the step itself.
const srbPreparedSteps = "const { When, Then } = require('@cucumber/cucumber');\n" +
	"const { spawnSync } = require('node:child_process');\n" +
	"const assert = require('node:assert');\n" +
	"const fs = require('node:fs');\n" +
	"const os = require('node:os');\n" +
	"const path = require('node:path');\n" +
	"When('is-even: I list the active accounts', function () {\n" +
	"  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'accounts-'));\n" +
	"  try {\n" +
	"    fs.mkdirSync(path.join(dir, 'db'));\n" +
	"    fs.writeFileSync(path.join(dir, 'db', 'accounts.json'), JSON.stringify({ accounts: ['alice', 'bob'], active: ['alice'] }));\n" +
	"    const r = spawnSync(process.execPath, [path.resolve('accounts.js'), '--active'], { cwd: dir, encoding: 'utf8' });\n" +
	"    this.out = r.stdout + r.stderr;\n" +
	"  } finally {\n" +
	"    fs.rmSync(dir, { recursive: true, force: true });\n" +
	"  }\n" +
	"});\n" +
	"Then('is-even: the output is {string}', function (want) {\n" +
	"  assert.strictEqual(this.out, want + '\\n');\n" +
	"});\n"

func srbEnvReview(id string) shpStep {
	return shpAnswer(shpCall(id, SubmitContractReviewTool, map[string]any{"findings": []any{}, "environment": []any{
		map[string]any{"scenario": "List active accounts", "output": "missing accounts database", "why": "the database is prepared only in the shaping checkout"},
	}}))
}

func srbConfirm(id, why string) shpStep {
	return shpAnswer(shpCall(id, SubmitContractConfirmationTool, map[string]any{"verdicts": []any{
		map[string]any{"finding": 1, "verdict": "confirmed", "why": why},
	}}))
}

// ---- the property journeys ----

const srbText = "exports.slug = (text) => text;\n"

const srbBound = "Its result is at most 20 characters long."

const srbSlugRequest = "Add slug(text) to text.js: it lowercases the text and joins words with dashes. " + srbBound

const srbSlugFeature = "Feature: Slugs\n  Scenario: Slugs join lowercase words\n    Then is-even: the slug of \"Hello World\" is \"hello-world\"\n"

const srbBoundScenario = "  Scenario: Slugs are at most 20 characters\n    Then is-even: the slug of a 30-letter text has at most 20 characters\n"

const srbSlugHeader = "const { Then } = require('@cucumber/cucumber');\n" +
	"const assert = require('node:assert/strict');\n" +
	"const { slug } = require('../../text');\n" +
	"Then('is-even: the slug of {string} is {string}', function (text, want) {\n" +
	"  assert.strictEqual(slug(text), want);\n" +
	"});\n"

const srbBoundSteps = srbSlugHeader +
	"Then('is-even: the slug of a {int}-letter text has at most {int} characters', function (n, max) {\n" +
	"  const out = slug('a'.repeat(n));\n" +
	"  assert.ok(out.length <= max, 'slug length ' + out.length + ' exceeds ' + max);\n" +
	"});\n"

const srbWeakSteps = srbSlugHeader +
	"Then('is-even: the slug of a {int}-letter text has at most {int} characters', function (n, max) {\n" +
	"  assert.strictEqual(typeof slug('a'.repeat(n)), 'string');\n" +
	"});\n"

var (
	srbSlugIntent    = srbIntent("Add a slug function", "slug(text) lowercases the text and joins words with dashes. "+srbBound)
	srbDeletedIntent = srbIntent("Add a slug function", "slug(text) lowercases the text and joins words with dashes.")
)

func srbPropertyReview(id string, scenarios ...string) shpStep {
	list := []any{}
	for _, s := range scenarios {
		list = append(list, s)
	}
	return shpAnswer(shpCall(id, SubmitContractReviewTool, map[string]any{"findings": []any{}, "environment": []any{}, "properties": []any{
		map[string]any{"property": "slug results are at most 20 characters", "request": srbBound, "kind": "new", "observable": true,
			"scenarios": list, "decision": "", "issues": []any{}},
	}}))
}

// ---- observation ----

// srbUserTexts are the user texts a request carries, in order, on the Responses or Messages wire.
func srbUserTexts(req map[string]any) []string {
	var out []string
	items := forceConversation(req)
	for _, it := range items {
		m, _ := it.(map[string]any)
		if m["role"] != "user" {
			continue
		}
		switch c := m["content"].(type) {
		case string:
			out = append(out, c)
		case []any:
			for _, b := range c {
				bm, _ := b.(map[string]any)
				if s, ok := bm["text"].(string); ok {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

func TestSRBAcceptance(t *testing.T) {
	wires := map[string]*forceWire{"responses": responsesWire, "anthropic": anthropicWire}
	w := wires[os.Getenv("SRB_WIRE")]
	if w == nil {
		t.Fatalf("unknown SRB_WIRE %q", os.Getenv("SRB_WIRE"))
	}
	p := w.prose
	submit := func(id string, args map[string]any) shpStep { return w.calls(shpCall(id, "submit", args)) }
	call := func(id, name string, args map[string]any) map[string]any { return shpCall(id, name, args) }
	readme := call("read", "read_file", map[string]any{"path": "README.md"})
	intentOnly := srbFiles(srbFile(srbIntentPath, shpScopedIntentMD))
	feature := srbFiles(srbFile(srbFeaturePath, shpFeature))
	both := map[string]any{"questions": []any{shpQuestion("Q?")}, "files": shpFiles()["files"]}
	request := "Add calc.is_even(n)."
	var f *shpFx
	var steps []shpStep
	var published map[string]string // the package a ready journey publishes
	switch script := os.Getenv("SRB_SCRIPT"); script {
	case "missing-feature":
		f = w.newShape(t)
		steps = []shpStep{
			w.calls(readme, call("c1", "submit", intentOnly)),
			p("The acceptance fixture still needs investigation."),
			w.calls(call("list", "list_files", map[string]any{"path": "features/steps"}), call("look", "read_file", map[string]any{"path": "calc.js"})),
			w.calls(call("add", "propose_files", srbFiles(srbFile(srbFeaturePath, shpFeature), srbFile(srbStepsPath, shpSteps)))),
			submit("c2", feature),
			submit("c3", feature),
		}
		published = map[string]string{srbIntentPath: shpScopedIntentMD, srbFeaturePath: shpFeature, srbStepsPath: shpSteps}
	case "observations-then-prose":
		f = w.newShape(t)
		steps = []shpStep{
			w.calls(readme, call("c1", "submit", shpFiles())),
			p("The failures need a look."),
			submit("c2", feature),
		}
		published = map[string]string{srbIntentPath: shpScopedIntentMD, srbFeaturePath: shpFeature, srbStepsPath: shpSteps}
	case "question-then-incomplete":
		f = w.newShape(t)
		steps = []shpStep{
			submit("c1", map[string]any{"questions": []any{shpQuestion("Which database library should store the accounts?", "SQLite", "A JSON file")}}),
			shpQuestionCheck("qc", "technical"),
			w.calls(readme, call("c2", "submit", intentOnly)),
			p("The acceptance fixture still needs investigation."),
			p("I still need to choose the database library."),
			p("I cannot finish the package.\nThe database is undecided."),
			p("unused"),
		}
	case "prose-and-refusals":
		f = w.newShape(t)
		steps = []shpStep{p("p1"), p("p2"), submit("c1", both), p("p3"), p("p4"), submit("c2", both), p("p5"), p("p6"), p("Last words.\nmore"), p("unused")}
	case "valid-first":
		f = w.newShape(t)
		f.setReport("PASSED", 0)
		steps = []shpStep{w.calls(readme, call("c1", "submit", shpFiles()))}
		published = map[string]string{srbIntentPath: shpScopedIntentMD, srbFeaturePath: shpFeature, srbStepsPath: shpSteps}
	case "ui-question-after-refusal":
		f = w.newShape(t)
		steps = []shpStep{
			w.calls(readme, call("c1", "submit", intentOnly)),
			p("The empty list needs a product decision."),
			submit("c2", map[string]any{"questions": []any{shpQuestion("Should an empty list show a message or stay blank?", "Show a message", "Stay blank")}}),
			shpQuestionCheck("qc", "ui_ux"),
		}
	case "unprepared-database", "unprepared-database-unchanged":
		f = newShpFxRun(t, "'true'")
		w.configure(f)
		f.Prov.contractWire = func(c map[string]any) shpStep { return w.calls(c) }
		f.write(".gitignore", ".kogen/local/\ndb/\n")
		f.write(".kogen/project.yaml", srbConfig)
		f.write("accounts.js", srbAccounts)
		f.write("features/steps/calc_steps.js", "'use strict';\n")
		f.git("add", "-A")
		f.git("commit", "-qm", "accounts")
		f.write("db/accounts.json", srbAccountsDB)
		request = "Add accounts.js --active to list only the active accounts."
		unprepared := srbFiles(srbFile(srbIntentPath, srbAccountsIntent), srbFile(srbFeaturePath, srbAccountsFeature), srbFile(srbStepsPath, srbUnpreparedSteps))
		if script == "unprepared-database" {
			steps = []shpStep{
				w.calls(readme, call("c1", "submit", unprepared)),
				p("The fresh copy has no prepared database; I need to inspect how the step can prepare it."),
				w.calls(call("look", "read_file", map[string]any{"path": "accounts.js"}), call("try", "probe", map[string]any{"command": "node accounts.js --active"})),
				w.calls(call("fix", "propose_files", srbFiles(srbFile(srbStepsPath, srbPreparedSteps)))),
				submit("c2", srbFiles(srbFile(srbStepsPath, srbPreparedSteps))),
				submit("c3", srbFiles(srbFile(srbFeaturePath, srbAccountsFeature))),
			}
			f.Prov.contract = []shpStep{
				srbEnvReview("r1"), srbConfirm("v1", "the fresh copy has no prepared database"),
				srbEnvReview("r2"),
			}
			published = map[string]string{srbIntentPath: srbAccountsIntent, srbFeaturePath: srbAccountsFeature, srbStepsPath: srbPreparedSteps}
		} else {
			steps = []shpStep{
				w.calls(readme, call("c1", "submit", unprepared)),
				p("The fresh copy has no prepared database."),
				submit("c2", unprepared),
				p("The package should be fine as it is."),
				submit("c3", unprepared),
				p("unused"),
			}
			f.Prov.contract = []shpStep{srbEnvReview("r1"), srbConfirm("v1", "the fresh copy has no prepared database")}
		}
	case "property-repaired", "property-weakened", "property-deleted":
		f = newShpFxRun(t, "'true'")
		w.configure(f)
		f.Prov.contractWire = func(c map[string]any) shpStep { return w.calls(c) }
		f.write(".kogen/project.yaml", srbConfig)
		f.write("text.js", srbText)
		f.write("features/steps/calc_steps.js", "'use strict';\n")
		f.git("add", "-A")
		f.git("commit", "-qm", "text")
		request = srbSlugRequest
		first := srbFiles(srbFile(srbIntentPath, srbSlugIntent), srbFile(srbFeaturePath, srbSlugFeature), srbFile(srbStepsPath, srbSlugHeader))
		covered := srbSlugFeature + srbBoundScenario
		steps = []shpStep{w.calls(readme, call("c1", "submit", first)), p("The length bound still has no scenario.")}
		switch script {
		case "property-repaired":
			steps = append(steps,
				w.calls(call("look", "read_file", map[string]any{"path": "text.js"})),
				w.calls(call("fix", "propose_files", srbFiles(srbFile(srbFeaturePath, covered), srbFile(srbStepsPath, srbBoundSteps)))),
				submit("c2", srbFiles(srbFile(srbStepsPath, srbBoundSteps))),
				submit("c3", srbFiles(srbFile(srbFeaturePath, covered))),
			)
			f.Prov.contract = []shpStep{srbPropertyReview("r1"), srbConfirm("v1", "no scenario covers the bound"), srbPropertyReview("r2", "Slugs are at most 20 characters")}
			published = map[string]string{srbIntentPath: srbSlugIntent, srbFeaturePath: covered, srbStepsPath: srbBoundSteps}
		case "property-weakened":
			steps = append(steps,
				w.calls(call("fix", "propose_files", srbFiles(srbFile(srbFeaturePath, covered), srbFile(srbStepsPath, srbWeakSteps)))),
				submit("c2", srbFiles(srbFile(srbStepsPath, srbWeakSteps))),
				p("The bound is covered now."),
				p("Nothing else to change."),
				p("Done.\nReally."),
				p("unused"),
			)
			f.Prov.contract = []shpStep{srbPropertyReview("r1"), srbConfirm("v1", "no scenario covers the bound"),
				srbPropertyReview("r2", "Slugs are at most 20 characters"), srbConfirm("v2", "the weakened step passes without the bound")}
		case "property-deleted":
			deleted := srbFiles(srbFile(srbIntentPath, srbDeletedIntent))
			steps = append(steps,
				submit("c2", deleted),
				p("The bound is gone from the Intent."),
				submit("c3", deleted),
				p("unused"),
			)
			f.Prov.contract = []shpStep{srbPropertyReview("r1"), srbConfirm("v1", "no scenario covers the bound"),
				srbPropertyReview("r2"), srbConfirm("v2", "the Request still asks for the bound")}
		}
	default:
		t.Fatalf("unknown SRB_SCRIPT %q", script)
	}
	f.Prov.seq = steps
	exit, err := f.shape(request)
	obs := map[string]any{"exit": exit, "stop": ""}
	if err != nil {
		if e, ok := sys.AsError(err); ok {
			obs["exit"], obs["stop"] = e.Exit, e.Render()
		} else {
			obs["exit"], obs["stop"] = -1, err.Error()
		}
	}
	reqs := shpShaperRequests(f.Prov.requests())
	forced, recovery := []int{}, []int{}
	messages := map[string]string{}
	seen := 0
	for i, r := range reqs {
		if !reflect.DeepEqual(r["tool_choice"], w.auto) {
			forced = append(forced, i+1)
		}
		var found []string
		for _, s := range srbUserTexts(r) {
			if strings.HasPrefix(s, srbRecovery) {
				found = append(found, s)
			}
		}
		if len(found) > seen {
			recovery = append(recovery, i+1)
			messages[fmt.Sprint(i+1)] = found[len(found)-1]
		}
		seen = len(found)
	}
	obs["requests"], obs["forced"], obs["recovery"], obs["messages"] = len(reqs), forced, recovery, messages
	outputs := map[string]string{}
	if len(reqs) > 0 {
		outputs = shpOutputs(reqs[len(reqs)-1])
	}
	obs["outputs"] = outputs
	events := []any{}
	for _, e := range shpNamed(f.events(), "recovery_turn") {
		delete(e, "event")
		events = append(events, e)
	}
	obs["recovery_events"] = events
	var mismatch []string
	if published != nil {
		paths := make([]string, 0, len(published))
		for p := range published {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		for _, p := range paths {
			if !f.exists(p) || f.read(p) != published[p] {
				mismatch = append(mismatch, p)
			}
		}
	}
	obs["published"] = published != nil && exit == 0 && err == nil && len(mismatch) == 0
	obs["unpublished"] = mismatch
	obs["printed"] = f.Stdout.String() + f.Stderr.String()
	b, err := json.Marshal(obs)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("SRB %s\n", b)
}
`;
