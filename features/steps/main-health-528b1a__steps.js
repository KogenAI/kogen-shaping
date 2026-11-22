'use strict';
// Runs a Go journey against this module's real Shape command with a scripted shaper. The Go source
// is written to a temporary file and compiled into the shape test binary with a go -overlay; it is
// never written into the checkout.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const {When, Then} = require('@cucumber/cucumber');

const TEST = 'TestMH528b1aBlockedNextLineAcceptance';
const NEXT = 'Next: revise the request, or review and commit .kogen/project.yaml outside Build, then shape again; preserve checks and actual check configuration. There is no protected-edit override.';
const NEW = 'so it needs no edit to the files listed above';
const GO = String.raw`package shape

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const mhNext = "Next: revise the request, or review and commit .kogen/project.yaml outside Build, then shape again; preserve checks and actual check configuration. There is no protected-edit override."

func TestMH528b1aBlockedNextLineAcceptance(t *testing.T) {
	var paths []string
	var reason, protect string
	switch mode := os.Getenv("MH_MODE"); mode {
	case "style":
		paths, reason, protect = []string{".style.cfg"}, "the requested check definition must change", "['.style.cfg']"
	case "glob":
		paths, reason, protect = []string{"settings/check.cfg"}, "the requested verification definition cannot be changed in Build", "['settings/*.cfg']"
	case "project":
		paths, reason = []string{".kogen/project.yaml"}, "the requested check definition must change"
	default:
		t.Fatalf("unknown MH_MODE %q", mode)
	}
	list := make([]any, len(paths))
	for i, p := range paths {
		list[i] = p
	}
	f := newShpFx(t, shpSubmit("c1", map[string]any{"blocked": map[string]any{"paths": list, "reason": reason}}))
	if protect != "" {
		for _, p := range paths {
			f.write(p, "x\n")
		}
		f.write(".kogen/project.yaml", "version: 1\ncommands:\n  checks: [{name: unit, run: 'true'}]\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\nprotect: "+protect+"\n")
		f.git("add", "-A")
		f.git("commit", "-qm", "Configure protection")
	}
	head := f.head()
	exit, err := f.shapeAs("blocked-next", "Change the protected files.")
	if err != nil || exit != 1 {
		t.Errorf("exit %d, error %v, want exit 1 and no error", exit, err)
	}
	want := "shape/protected_conflict: required edits touch protected files: " + strings.Join(paths, ", ") + "\n" + reason + "\n" + mhNext + "\n"
	if got := f.Stdout.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if f.exists(".kogen/intents/blocked-next") || f.exists(".kogen/local/approvals/blocked-next.json") || f.exists(".kogen/local/runs") {
		t.Error("a blocked report wrote an Intent, an approval or a run")
	}
	if f.head() != head {
		t.Error("a blocked report moved HEAD")
	}
	if !t.Failed() {
		fmt.Println("MH-OK")
	}
}
`;

let built = null;
function envFor(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'main-health-528b1a-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const root = process.cwd();
  const src = path.join(dir, 'journey_test.go');
  fs.writeFileSync(src, GO);
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.join(root, 'shape/mh528b1a_blocked_next_line_acceptance_test.go')]: src}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: envFor({}), encoding: 'utf8', timeout: 100000});
  built = {bin, error: r.error || r.status !== 0 ? `${TEST}: go test -c -vet=off -overlay ./shape failed (exit ${r.status}): ${r.error || ''}\n${r.stdout}${r.stderr}` : ''};
  return built;
}
function runJourney(mode) {
  const b = build();
  if (b.error) return {ok: false, detail: b.error};
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=100s'];
  const r = spawnSync(b.bin, args, {cwd: path.join(process.cwd(), 'shape'), env: envFor({MH_MODE: mode}), encoding: 'utf8', timeout: 110000, maxBuffer: 8 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  const ran = text.includes('--- PASS: ' + TEST + ' ');
  const ok = !r.error && r.status === 0 && ran && text.includes('MH-OK\n') && !text.includes('--- SKIP:');
  return {ok, detail: `${TEST} (MH_MODE=${mode}): ${args.join(' ')}; exit=${r.status}; error=${r.error || ''}; absent, skipped or zero runs=${!ran}\n${text}`};
}

When('main-health-528b1a: the blocked report for {string} runs', {timeout: 115000}, function (mode) {
  this.mhResult = runJourney(mode);
});
Then('main-health-528b1a: the blocked report prints the Next line naming the project definition', function () {
  assert.ok(this.mhResult.ok, this.mhResult.detail);
});

const REPAIR = 'features/steps/k-shape-blocked-report-repair__steps.js';
When('main-health-528b1a: the repair feature journeys run', {timeout: 115000}, function () {
  const root = process.cwd();
  this.mhRepair = [];
  for (const mode of ['mixed', 'protected']) {
    const r = spawnSync(process.execPath, [path.join(root, REPAIR), mode, '--diagnostic'], {cwd: root, env: envFor({}), encoding: 'utf8', timeout: 50000, maxBuffer: 8 * 1024 * 1024});
    this.mhRepair.push({mode, status: r.status, error: r.error ? String(r.error) : '', out: (r.stdout || '').trim(), err: (r.stderr || '').slice(-3000)});
  }
  const src = fs.readFileSync(path.join(root, REPAIR), 'utf8');
  this.mhConst = src.split('\n').find((l) => l.startsWith('const kbrNext = ')) || '';
});
Then('main-health-528b1a: the repair feature journeys passed with the project definition Next line', function () {
  assert.equal(this.mhConst, 'const kbrNext = "' + NEXT + '\\n"', 'the kbrNext constant of ' + REPAIR + ' does not hold the Next line naming the project definition');
  for (const r of this.mhRepair) {
    assert.ok(r.status === 0 && r.out === 'accepted', `${REPAIR} ${r.mode}: exit ${r.status}, ${r.error}, stdout ${JSON.stringify(r.out)}\n${r.err}`);
  }
});

When('main-health-528b1a: the Go sources are read', function () {
  this.mhSources = {};
  for (const f of ['shape/shape.go', 'shape/shape_test.go', 'shape/contract_test.go']) {
    this.mhSources[f] = fs.readFileSync(path.join(process.cwd(), f), 'utf8');
  }
});
Then('main-health-528b1a: the Go sources hold only the Next line naming the project definition', function () {
  for (const [f, text] of Object.entries(this.mhSources)) {
    assert.ok(text.includes(NEXT), f + ' does not hold the Next line naming .kogen/project.yaml');
    assert.ok(!text.includes(NEW), f + ' still holds the sentence "' + NEW + '"');
  }
});
