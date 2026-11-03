const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const cp = require('node:child_process');
const names = ['TestContractCheckRefusesAConfirmedContradictionAndPublishesItsCorrection', 'TestContractCheckExternalShape'];
const command = ['test', '-json', './shape', '-run', names.join('|'), '-count=1'];
function invoke(args, options = {}) {
  const r = cp.spawnSync('go', args, {encoding: 'utf8', ...options});
  assert.equal(r.status, 0, `go ${args.join(' ')}: ${r.error || ''}\n${r.stdout}\n${r.stderr}`);
  return r.stdout;
}
function capture() {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'shaping-proof-'));
  try {
    const mod = invoke(['list', '-m', '-f', '{{.Dir}}', 'github.com/KogenAI/kogen-verification']).trim();
    const copy = path.join(tmp, 'verification');
    fs.cpSync(mod, copy, {recursive: true});
    const original = path.join(copy, 'verify/report.go');
    let text = fs.readFileSync(original, 'utf8');
    const signature = 'func Account(root, feature string, expected []project.Scenario, report string, status int, output string) Acceptance {';
    assert.equal(text.split(signature).length, 2, 'Account capture boundary must be unique');
    // Capture only; execute the real verdict unchanged in a disposable module copy.
    text = text.replace(signature, signature + `
 if d := os.Getenv("KOGEN_SHAPING_PROOF_CAPTURE"); d != "" {
   b, e := os.ReadFile(report); if e != nil { panic(e) }
   v, e := json.Marshal(map[string]any{"expected": expected, "report": string(b)}); if e != nil { panic(e) }
   f, e := os.CreateTemp(d, "proof-*.json"); if e != nil { panic(e) }
   if _, e = f.Write(v); e != nil { panic(e) }; if e = f.Close(); e != nil { panic(e) }
 }
`);
    fs.chmodSync(original, 0o644);
    fs.writeFileSync(original, text);
    const modfile = path.join(tmp, 'capture.mod');
    fs.copyFileSync('go.mod', modfile);
    fs.copyFileSync('go.sum', path.join(tmp, 'capture.sum'));
    invoke(['mod', 'edit', '-modfile', modfile, '-replace', `github.com/KogenAI/kogen-verification=${copy}`]);
    const out = invoke(['test', '-modfile', modfile, ...command.slice(1)], {env: {...process.env, KOGEN_SHAPING_PROOF_CAPTURE: tmp}});
    const events = out.trim().split('\n').map(x => JSON.parse(x));
    for (const name of names) {
      const results = events.filter(e => e.Test === name && ['pass', 'fail', 'skip'].includes(e.Action));
      assert.equal(results.length, 1, `${name}: go ${command.join(' ')}: absent or repeated outcome`);
      assert.equal(results[0].Action, 'pass', `${name}: go ${command.join(' ')}: ${results[0].Action}`);
    }
    const records = fs.readdirSync(tmp).filter(x => /^proof-.*\.json$/.test(x)).map(x => JSON.parse(fs.readFileSync(path.join(tmp, x), 'utf8')));
    assert.ok(records.length > 0, `go ${command.join(' ')} captured no Account calls`);
    return records;
  } finally { fs.rmSync(tmp, {recursive: true, force: true}); }
}
function field(o, name) { return o[name] === undefined ? o[name[0].toLowerCase() + name.slice(1)] : o[name]; }
function validate(record) {
  const messages = record.report.trim().split('\n').map(x => JSON.parse(x));
  const sources = new Map();
  function visit(v) {
    if (!v || typeof v !== 'object') return;
    if (v.id && v.keyword !== undefined && v.location) sources.set(v.id, v);
    for (const x of Object.values(v)) if (typeof x === 'object') visit(x);
  }
  for (const m of messages) if (m.gherkinDocument) visit(m.gherkinDocument);
  const pickles = messages.filter(m => m.pickle).map(m => m.pickle);
  const expected = record.expected;
  assert.equal(pickles.length, expected.length, 'candidate scenario proof count');
  for (let i = 0; i < expected.length; i++) {
    const sc = expected[i];
    const pickle = pickles[i];
    assert.equal(pickle.name, field(sc, 'Name'), 'candidate scenario name');
    const actual = pickle.steps.map(s => {
      const source = sources.get(s.astNodeIds[0]);
      assert.ok(source, 'pickle step source must exist');
      return {Line: source.location.line, Keyword: source.keyword, Text: s.text};
    });
    const wanted = field(sc, 'Steps').map(s => ({Line: field(s, 'Line'), Keyword: field(s, 'Keyword'), Text: field(s, 'Text')}));
    assert.deepEqual(actual, wanted, `candidate step proof for ${pickle.name}`);
  }
}
if (require.main === module) {
  const records = capture();
  const good = records.find(r => { try { validate(r); return true; } catch { return false; } });
  assert.ok(good, 'no real internally consistent report for observer control');
  if (process.argv[2] === 'good') { validate(good); console.log('control accepted'); }
  else if (process.argv[2] === 'bad') {
    const bad = JSON.parse(JSON.stringify(good));
    const steps = field(bad.expected[0], 'Steps');
    const key = steps[0].Text === undefined ? 'text' : 'Text';
    steps[0][key] += ' stale candidate';
    try { validate(bad); console.log('control accepted'); } catch { console.log('control rejected'); process.exitCode = 1; }
  } else { throw new Error('expected good or bad'); }
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('I run the existing contract correction and external Shape regressions with proof capture', {timeout: 120000}, function () { this.shapingProofRecords = capture(); });
  Then('all captured candidate reports prove their exact expected steps', function () { for (const record of this.shapingProofRecords) validate(record); });
}
