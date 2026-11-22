'use strict';
// Step definitions of the Intent shape-invented-fix-names-kind-remedy: the fix line and the refusal
// line that the shape package gives a refused invented literal, observed through a temporary
// package-internal Go test that prints both lines for each literal as JSON.
const { When, Then } = require('@cucumber/cucumber');
const assert = require('node:assert/strict');
const cp = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const root = path.resolve(__dirname, '..', '..');
const PREFIX = 'shape-invented-fix-names-kind-remedy: ';
const probeName = 'zz_kogen_invented_fix_probe_test.go';

const probeSource = `package shape

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/KogenAI/kogen-coordinator/git"
)

func TestZZKogenInventedFixProbe(t *testing.T) {
	var literals []string
	if err := json.Unmarshal([]byte(os.Getenv("KOGEN_PROBE_LITERALS")), &literals); err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	out := map[string]map[string]string{}
	for _, l := range literals {
		sh := &shaping{repo: git.New(empty, os.Environ())}
		ev := contractEvidence{featurePath: "f.feature", expectations: map[string]string{"f.feature": "Then it uses " + l}}
		kept := sh.auditInventedKept(ev, []auditInventedReport{{"S", l, "W"}}, "")
		if len(kept) != 1 {
			t.Fatalf("literal %q: %d findings kept, want 1", l, len(kept))
		}
		out[l] = map[string]string{"line": kept[0].line, "fix": kept[0].fix, "direct": auditInventedFix(l)}
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("KOGEN_PROBE_OUT"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}
`;

const LITERALS = [
  'test/acceptance/external-stall-outcome.test.cjs',
  'alias Studio.Loop.Fairness',
  'hard_quota',
  'The table has one column each for Shape, Build and Review',
];

let observed = null;

function observe() {
  if (observed) return observed;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'kogen-fix-probe-'));
  const outFile = path.join(dir, 'out.json');
  const probe = path.join(root, 'shape', probeName);
  fs.writeFileSync(probe, probeSource);
  const command = 'go test -count=1 -run ^TestZZKogenInventedFixProbe$ ./shape';
  let r;
  try {
    r = cp.spawnSync('go', ['test', '-count=1', '-run', '^TestZZKogenInventedFixProbe$', './shape'], {
      cwd: root,
      encoding: 'utf8',
      timeout: 110000,
      env: { ...process.env, KOGEN_PROBE_OUT: outFile, KOGEN_PROBE_LITERALS: JSON.stringify(LITERALS) },
    });
  } finally {
    fs.rmSync(probe, { force: true });
  }
  assert.ok(r.status === 0 && fs.existsSync(outFile), `${command} (in ${root}) exited ${r.status}${r.error ? ` (${r.error.message})` : ''}:\n${r.stdout}${r.stderr}`);
  observed = JSON.parse(fs.readFileSync(outFile, 'utf8'));
  fs.rmSync(dir, { recursive: true, force: true });
  return observed;
}

function entry(literal) {
  const e = observe()[literal];
  assert.ok(e, `no observation for the literal ${literal}`);
  return e;
}

When(PREFIX + 'the shape package gives each of the four invented literals its fix line and refusal line', { timeout: 120000 }, function () {
  observe();
});

Then(PREFIX + 'the fix line for {string} contains {string}', function (literal, text) {
  const fix = entry(literal).fix;
  assert.ok(fix.includes(text), `the fix line for ${literal} lacks ${JSON.stringify(text)}: ${fix}`);
});

Then(PREFIX + 'the fix line for {string} does not contain {string}', function (literal, text) {
  const fix = entry(literal).fix;
  assert.ok(!fix.includes(text), `the fix line for ${literal} contains ${JSON.stringify(text)}: ${fix}`);
});

Then(PREFIX + 'the fix line for {string} is the fix line for {string} with its own literal', function (literal, other) {
  const a = entry(literal).fix;
  const b = entry(other).fix;
  assert.equal(a.replace(`"${literal}"`, '"L"'), b.replace(`"${other}"`, '"L"'));
});

Then(PREFIX + 'the fix line for {string} is byte for byte:', function (literal, doc) {
  const e = entry(literal);
  assert.equal(e.fix, doc);
  assert.equal(e.direct, doc);
});

Then(PREFIX + 'the refusal line for {string} is byte for byte:', function (literal, doc) {
  assert.equal(entry(literal).line, doc);
});
