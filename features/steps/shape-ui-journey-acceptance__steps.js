'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const assert = require('node:assert/strict');
const { When, Then } = require('@cucumber/cucumber');
const TEST = 'TestShapeUIJourneyGuidance';
const GO = String.raw`package shape

import (
	"encoding/json"
	"os"
	"testing"
)

func TestShapeUIJourneyGuidance(t *testing.T) {
	f := newShpFx(t, shpSubmit("q1", map[string]any{"questions": []any{shpQuestion("Which name?", "a", "b")}}), shpUI(1))
	if _, err := f.shape(os.Getenv("KG_REQUEST")); err != nil {
		t.Fatal(err)
	}
	reqs := f.Prov.requests()
	if len(reqs) == 0 {
		t.Fatal("no shaper request")
	}
	ins, _ := reqs[0]["instructions"].(string)
	b, e := json.Marshal(map[string]any{"shaper": ins, "review": contractReviewInstructions})
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(os.Getenv("KG_OUT"), b, 0600); e != nil {
		t.Fatal(e)
	}
}
`;
const env = (extra = {}) => { const e = { ...process.env, ...extra }; for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k]; return e; };
let built;
When('shape-ui-journey-acceptance: Shape processes {string}', function (request) {
  if (!built) {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-journey-'));
    process.on('exit', () => fs.rmSync(dir, { recursive: true, force: true }));
    const src = path.join(dir, 'journey_test.go');
    fs.writeFileSync(src, GO);
    const overlay = path.join(dir, 'overlay.json');
    fs.writeFileSync(overlay, JSON.stringify({ Replace: { 'shape/shape_ui_journey_acceptance_test.go': src } }));
    const bin = path.join(dir, 'shape.test');
    const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], { env: env(), encoding: 'utf8', timeout: 200000 });
    assert.equal(r.status, 0, TEST + ': go test -c -overlay ./shape exit=' + r.status + '\n' + r.stdout + r.stderr);
    built = { dir, bin };
  }
  const out = path.join(built.dir, 'out' + Date.now() + '.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.timeout=110s'];
  const r = spawnSync(built.bin, args, { cwd: path.resolve('shape'), env: env({ KG_OUT: out, KG_REQUEST: request }), encoding: 'utf8', timeout: 115000 });
  assert.ok(r.status === 0 && fs.existsSync(out), TEST + ': shape.test ' + args.join(' ') + ' exit=' + r.status + '; test absent, skipped or failed if no observation\n' + r.stdout + r.stderr);
  this.obs = JSON.parse(fs.readFileSync(out, 'utf8'));
});
function states(text, phrase, who) {
  assert.ok(text.includes(phrase), who + ' do not state ' + JSON.stringify(phrase));
}
Then('shape-ui-journey-acceptance: the shaper instructions state {string}', function (phrase) { states(this.obs.shaper, phrase, 'the shaper instructions'); });
Then('shape-ui-journey-acceptance: the contract review instructions state {string}', function (phrase) { states(this.obs.review, phrase, 'the contract review instructions'); });
