'use strict';
// Acceptance of shape-tracked-directory-evidence-no-slash. The Given step compiles the shape
// package's tests with an added Go test file (through a Go overlay), runs it once per Cucumber
// process and reads the JSON observation of strings it wrote to STDE_OUT. The Then steps compare
// one observation with the feature's value.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const TEST = 'TestShapeTrackedDirectoryEvidenceNoSlashAcceptance';
const TARGET = 'shape/shape_tracked_directory_evidence_no_slash_acceptance_test.go';
const P = 'shape-tracked-directory-evidence-no-slash: ';

const GO = `package shape

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestShapeTrackedDirectoryEvidenceNoSlashAcceptance(t *testing.T) {
	f := newShpFx(t)
	f.write("docs/presets/guide.md", "Tracked preset configuration.\\n")
	f.write("docs/README.md", "The repository documentation.\\n")
	f.git("add", "docs")
	f.git("commit", "-qm", "Add tracked documentation and presets")
	f.write("docs/drafts/new.md", "Not committed.\\n")
	f.newSession()
	obs := map[string]string{}
	fail := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct{ key, cite string }{{"bare", "docs/presets"}, {"slash", "docs/presets/"}} {
		sh := f.shaping()
		sh.listedPaths = map[string]bool{"docs": true}
		intent := "## Decisions\\n- Preserve the presets. (evidence: " + c.cite + ") (source: request)\\n"
		p, err := sh.evidenceProblems(intent)
		fail(err)
		obs[c.key+"_problems"] = strings.Join(p, "\\n")
		ev, err := sh.decisionEvidence(intent)
		fail(err)
		obs[c.key+"_context_has_guide"] = strconv.FormatBool(strings.Contains(ev, "guide.md") && !strings.Contains(ev, "not observed"))
		_, _, diag, err := sh.decisionReceipt(1, c.cite)
		fail(err)
		obs[c.key+"_ledger"] = diag
		q, _, err := sh.citedEvidence([]map[string]any{{"question": "Which? (evidence: " + c.cite + ")"}})
		fail(err)
		obs[c.key+"_question"] = q
	}
	sh := f.shaping()
	sh.listedPaths = map[string]bool{"docs": true}
	p, err := sh.evidenceProblems("## Decisions\\n- Keep the readme. (evidence: docs/README.md) (source: request)\\n")
	fail(err)
	obs["unread_file"] = strings.Join(p, "\\n")
	p, err = sh.evidenceProblems("## Decisions\\n- Keep drafts. (evidence: docs/drafts) (source: request)\\n")
	fail(err)
	obs["untracked_dir"] = strings.Join(p, "\\n")
	q, _, err := sh.citedEvidence([]map[string]any{{"question": "Which? (evidence: docs/drafts)"}})
	fail(err)
	obs["untracked_question"] = q
	sh = f.shaping()
	p, err = sh.evidenceProblems("## Decisions\\n- Keep presets. (evidence: docs/presets) (source: request)\\n")
	fail(err)
	obs["unlisted_dir"] = strings.Join(p, "\\n")
	data, err := json.Marshal(obs)
	fail(err)
	fail(os.WriteFile(os.Getenv("STDE_OUT"), data, 0o644))
}
`;

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

let observation = null;
function observe() {
  if (observation !== null) return observation;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'stde-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, GO);
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({STDE_OUT: out}), encoding: 'utf8', timeout: 280000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout}${r.stderr}`;
  assert.equal(r.status, 0, shown);
  const passes = (r.stdout || '').split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' ')).length;
  assert.equal(passes, 1, `${TEST} did not pass exactly once: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  observation = JSON.parse(fs.readFileSync(out, 'utf8'));
  return observation;
}

Given(P + 'a Git fixture with tracked docs\\/presets files, an uncommitted docs\\/drafts directory and the directory docs listed', function () {
  observe();
});

Then(P + 'the observation {string} is {string}', function (name, want) {
  const o = observe();
  assert.ok(Object.hasOwn(o, name), `no observation "${name}" in ${JSON.stringify(o, null, 1)}`);
  assert.equal(o[name], want, `observation "${name}" is ${JSON.stringify(o[name])}, want ${JSON.stringify(want)}`);
});
