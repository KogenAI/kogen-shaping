'use strict';
// Steps of k-shape-drop-optins. kogen-shaping must build, vet and pass its tests against a
// coordinator that rejects the removed opt-in keys. That coordinator is synthesized from the pinned
// one in a temporary folder: each removed project.yaml key's case label becomes unknown (a file that
// sets it is refused environment/project_config_invalid) and each removed exported identifier
// becomes unexported (no other module can read it). The behaviour cases run the overlay test
// k_shape_drop_optins_test.go.txt against the pinned coordinator, which still parses the keys.
//   node features/steps/k_shape_drop_optins_steps.js good|bad                   observer controls of the synthesis
//   node features/steps/k_shape_drop_optins_steps.js overlay-good|overlay-bad   observer controls of the overlay
const { spawnSync } = require('node:child_process');
const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');

const COORDINATOR = 'github.com/KogenAI/kogen-coordinator';
const REMOVED_IDENTIFIERS = ['MaxRounds', 'DefaultExternalRounds', 'ShapeWorkers', 'Workers', 'ShapeAudit', 'Audit', 'ShapeDecisionCheck', 'DecisionCheck', 'ReviewMaterial'];
const REMOVED_KEYS = ['shape_workers', 'shape_audit', 'shape_decision_check', 'review_material', 'max_rounds'];
const LIMIT = 40 * 60 * 1000;

function run(cmd, args, opts = {}) {
  const r = spawnSync(cmd, args, { encoding: 'utf8', timeout: LIMIT, maxBuffer: 64 * 1024 * 1024, ...opts });
  const status = r.error ? `error ${r.error.message}` : r.signal ? `signal ${r.signal}` : `exit ${r.status}`;
  return { ok: r.status === 0 && !r.error, status, stdout: r.stdout || '', out: (r.stdout || '') + (r.stderr || '') };
}

// pinnedCoordinator is the module folder of the coordinator root's go.mod pins.
function pinnedCoordinator(root) {
  const r = run('go', ['list', '-m', '-json', COORDINATOR], { cwd: root });
  if (!r.ok) throw new Error(`go list -m -json ${COORDINATOR}: ${r.status}\n${r.out}`);
  const dir = JSON.parse(r.stdout).Dir;
  if (!dir || !fs.existsSync(path.join(dir, 'project', 'config.go'))) throw new Error(`no pinned coordinator source: ${r.stdout}`);
  return dir;
}

// synthesize writes the pinned coordinator, rejecting the removed keys and hiding the removed
// identifiers, to dir.
function synthesize(root, dir) {
  const ident = new RegExp(`\\b(${REMOVED_IDENTIFIERS.join('|')})\\b`, 'g');
  const label = new RegExp(`case "(${REMOVED_KEYS.join('|')})":`, 'g');
  let hidden = 0;
  const rejected = new Set();
  const copy = (from, to) => {
    fs.mkdirSync(to, { recursive: true });
    for (const e of fs.readdirSync(from, { withFileTypes: true })) {
      const a = path.join(from, e.name);
      const b = path.join(to, e.name);
      if (e.isDirectory()) {
        copy(a, b);
      } else if (e.name.endsWith('.go') && !e.name.endsWith('_test.go')) {
        const text = fs.readFileSync(a, 'utf8');
        const next = text.replace(label, (_, key) => { rejected.add(key); return `case "removed:${key}":`; }).replace(ident, 'removed$1');
        if (next !== text) hidden++;
        fs.writeFileSync(b, next);
      } else if (e.isFile()) {
        fs.writeFileSync(b, fs.readFileSync(a));
      }
    }
  };
  copy(pinnedCoordinator(root), dir);
  const missing = REMOVED_KEYS.filter((k) => !rejected.has(k));
  if (hidden === 0 || missing.length) throw new Error(`the pinned coordinator does not parse every removed key: missing ${missing.join(', ')}`);
}

// goChecks runs go build, go vet and go test -count=1 of every package of the module at dir
// against the coordinator at coordinator (a replace in a temporary copy of its go.mod), stopping
// at the first failure. It returns the last command run: the go test run when all three pass.
function goChecks(dir, coordinator) {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'kso-mod-'));
  try {
    const modfile = path.join(tmp, 'go.mod');
    fs.copyFileSync(path.join(dir, 'go.mod'), modfile);
    fs.copyFileSync(path.join(dir, 'go.sum'), path.join(tmp, 'go.sum'));
    const edit = run('go', ['mod', 'edit', `-replace=${COORDINATOR}=${coordinator}`, modfile], { cwd: dir });
    if (!edit.ok) throw new Error(`go mod edit: ${edit.status}\n${edit.out}`);
    const used = run('go', ['list', '-modfile=' + modfile, '-m', '-json', COORDINATOR], { cwd: dir });
    if (!used.ok || (JSON.parse(used.stdout).Replace || {}).Path !== coordinator) {
      throw new Error(`the synthetic coordinator is not the one in use: ${used.status}\n${used.out}`);
    }
    let last;
    for (const [verb, ...rest] of [['build', './...'], ['vet', './...'], ['test', '-count=1', './...']]) {
      const args = [verb, '-modfile=' + modfile, ...rest];
      const r = run('go', args, { cwd: dir });
      last = { ok: r.ok, command: `go ${args.join(' ')}`, status: r.status, out: r.out };
      if (!r.ok) break;
    }
    return last;
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

// fixture writes a small module that uses the coordinator as kogen-shaping's go.mod pins it.
function fixture(root, dir, source, test) {
  fs.mkdirSync(dir, { recursive: true });
  const mod = fs.readFileSync(path.join(root, 'go.mod'), 'utf8').replace(/^module .*$/m, 'module ksofixture');
  fs.writeFileSync(path.join(dir, 'go.mod'), mod);
  fs.copyFileSync(path.join(root, 'go.sum'), path.join(dir, 'go.sum'));
  fs.writeFileSync(path.join(dir, 'fixture.go'), `package fixture\n\nimport "${COORDINATOR}/project"\n\n${source}\n`);
  fs.writeFileSync(path.join(dir, 'fixture_test.go'), `package fixture\n\nimport (\n\t"testing"\n\n\t"${COORDINATOR}/project"\n)\n\n${test}\n`);
}

const parses = (yaml) => `func TestParses(t *testing.T) {\n\tif _, err := project.ParseConfig(${JSON.stringify(yaml)}, "f"); err != nil {\n\t\tt.Fatal(err)\n\t}\n}`;
const kept = 'func Kept(c *project.Config) (int, string, string) {\n\tr := c.Models["shape"]\n\treturn len(c.Checks), r.Executor, r.Model\n}';
const goodFixture = {
  source: kept,
  test: 'func TestKept(t *testing.T) {\n\tc, err := project.ParseConfig("version: 1\\nmodels:\\n  shape: {executor: claude-code, model: s, effort: high}\\n", "f")\n\tif err != nil {\n\t\tt.Fatal(err)\n\t}\n\tif n, e, m := Kept(c); n != 0 || e != "claude-code" || m != "s" {\n\t\tt.Fatal(n, e, m)\n\t}\n}',
};
const badFixtures = {
  'Role.MaxRounds': 'var _ = project.Role{}.MaxRounds',
  DefaultExternalRounds: 'var _ = project.DefaultExternalRounds',
  'Config.ShapeWorkers': 'var _ = project.Config{}.ShapeWorkers',
  Workers: 'var _ project.Workers',
  'Config.ShapeAudit': 'var _ = project.Config{}.ShapeAudit',
  Audit: 'var _ project.Audit',
  'Config.ShapeDecisionCheck': 'var _ = project.Config{}.ShapeDecisionCheck',
  DecisionCheck: 'var _ project.DecisionCheck',
  'Config.ReviewMaterial': 'var _ = project.Config{}.ReviewMaterial',
  ReviewMaterial: 'var _ project.ReviewMaterial',
  'models.shape.max_rounds': parses('version: 1\nmodels:\n  shape: {executor: claude-code, model: s, effort: high, max_rounds: 2}\n'),
  shape_workers: parses('version: 1\nshape_workers:\n  enabled: false\n'),
  shape_audit: parses('version: 1\nshape_audit:\n  enabled: false\n'),
  shape_decision_check: parses('version: 1\nshape_decision_check:\n  enabled: false\n'),
  review_material: parses('version: 1\nreview_material:\n  mode: parts\n'),
};

// control is the observer control: the known-good fixture passes against the synthetic
// coordinator, and each known-bad fixture (one removed identifier or key each) passes against the
// pinned coordinator and fails against the synthetic one.
function control(mode) {
  const root = process.cwd();
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'kso-control-'));
  try {
    const coordinator = path.join(tmp, 'coordinator');
    synthesize(root, coordinator);
    if (mode === 'good') {
      fixture(root, path.join(tmp, 'good'), goodFixture.source, goodFixture.test);
      const r = goChecks(path.join(tmp, 'good'), coordinator);
      return r.ok ? 'observer accepted' : `observer rejected ${r.command}: ${r.status}\n${r.out}`;
    }
    const pinned = pinnedCoordinator(root);
    for (const [name, body] of Object.entries(badFixtures)) {
      const dir = path.join(tmp, 'bad-' + name.replace(/\W/g, '-'));
      const isTest = body.startsWith('func Test');
      fixture(root, dir, isTest ? kept : body, isTest ? body : 'func TestUses(t *testing.T) {\n\t_ = project.Config{}\n}');
      const p = goChecks(dir, pinned);
      if (!p.ok) return `observer error: the ${name} fixture fails against the pinned coordinator, ${p.command}: ${p.status}\n${p.out}`;
      if (goChecks(dir, coordinator).ok) return `observer accepted ${name}`;
    }
    return 'observer rejected';
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

// overlayRun runs one test of the overlay at source (the overlay file by default) against the
// pinned coordinator with `go test -json`: whether it passed, its own assertion lines and the full
// account, which keeps the run's JSON rows so that a failure shows whether the test ran.
function overlayRun(name, env, source = path.resolve('features/steps/k_shape_drop_optins_test.go.txt')) {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'kso-overlay-'));
  try {
    const overlay = path.join(tmp, 'overlay.json');
    fs.writeFileSync(overlay, JSON.stringify({ Replace: { [path.resolve('shape/k_shape_drop_optins_acceptance_test.go')]: source } }));
    const args = ['test', '-json', '-overlay', overlay, './shape', '-run', `^${name}$`, '-count=1'];
    // Go matches overlay paths against its working directory as $PWD names it.
    const r = run('go', args, { cwd: process.cwd(), env: { ...process.env, ...env, PWD: process.cwd() } });
    const what = `go ${args.join(' ')} (${Object.entries(env).map(([k, v]) => `${k}=${v}`).join(' ')}): ${r.status}`;
    let ran = false;
    let verdict = '';
    const output = [];
    for (const line of r.out.split('\n')) {
      let row;
      try {
        row = JSON.parse(line);
      } catch {
        continue;
      }
      if (!row || row.Test !== name) continue;
      if (row.Action === 'run') ran = true;
      else if (['pass', 'fail', 'skip'].includes(row.Action)) verdict = row.Action;
      else if (row.Action === 'output') output.push(row.Output || '');
    }
    const assertions = output.join('').split('\n').filter((l) => /^\s+k_shape_drop_optins_acceptance_test\.go:\d+: /.test(l)).map((l) => l.trim());
    const state = verdict ? `${name}: ${verdict}` : ran ? `${name}: ran without a verdict` : `${name} did not run (absent, skipped or run zero times)`;
    const passed = r.ok && verdict === 'pass';
    return { passed, assertions, text: `${state}; ${assertions.length ? assertions.join('; ') : 'no assertion output'}; ${what}\n${r.out}` };
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

function overlayTest(name, env) {
  const r = overlayRun(name, env);
  if (!r.passed) throw new Error(r.text);
}

// defaultsVariant writes the overlay with its fixtures at today's defaults (no opt-in key and no
// max_rounds) to dir: what a correct implementation does with the opt-ins, the unchanged tree does
// without them, so the overlay's deeper assertions are qualified independently of the red they mask.
function defaultsVariant(dir) {
  const text = fs.readFileSync('features/steps/k_shape_drop_optins_test.go.txt', 'utf8');
  const next = text.replace(/const kdoOptins = [\s\S]*?mode: on_demand\\n"\n/, 'const kdoOptins = "# none\\n"\n').replace(', max_rounds: 1}', '}');
  if (!next.includes('const kdoOptins = "# none\\n"') || next.includes('shape_workers:\\n') || next.includes('max_rounds: 1')) throw new Error('the overlay has no opt-in block or max_rounds line to remove');
  const file = path.join(dir, 'defaults.go.txt');
  fs.writeFileSync(file, next);
  return file;
}

// overlayControl is the observer control of the overlay. good: the defaults variant passes every
// case. bad: the defaults variant with a wrong expected round cap fails at that assertion.
function overlayControl(mode) {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'kso-variant-'));
  try {
    const source = defaultsVariant(tmp);
    if (mode === 'overlay-good') {
      for (const [name, env] of [['TestKShapeDropOptinsNative', {}],
        ['TestKShapeDropOptinsExternal', { KDO_CASE: 'max_rounds', KDO_CAP: '3', KDO_WANT: '3' }],
        ['TestKShapeDropOptinsExternal', { KDO_CASE: 'max_rounds', KDO_CAP: '0', KDO_WANT: '6' }],
        ['TestKShapeDropOptinsExternal', { KDO_CASE: 'opt-ins', KDO_CAP: '6', KDO_WANT: '6' }]]) {
        const r = overlayRun(name, env, source);
        if (!r.passed) return `observer rejected ${r.text}`;
      }
      return 'observer accepted';
    }
    const r = overlayRun('TestKShapeDropOptinsExternal', { KDO_CASE: 'max_rounds', KDO_CAP: '3', KDO_WANT: '4' }, source);
    if (r.passed) return 'observer accepted a wrong round cap';
    if (!r.assertions.some((a) => a.endsWith("k-shape-drop-optins: the session's round cap is 3, want 4 (executor shape cap 3)"))) return `observer error: not rejected at the round cap assertion: ${r.text}`;
    return 'observer rejected';
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

// canonical is a JSON value with every object's keys sorted, so that goldens compare semantically.
function canonical(v) {
  if (Array.isArray(v)) return `[${v.map(canonical).join(',')}]`;
  if (v && typeof v === 'object') return `{${Object.keys(v).sort().map((k) => `${JSON.stringify(k)}:${canonical(v[k])}`).join(',')}}`;
  return JSON.stringify(v);
}
const sha256 = (s) => crypto.createHash('sha256').update(s).digest('hex');
const golden = (name) => JSON.parse(fs.readFileSync(path.join('shape', 'testdata', name), 'utf8'));

// The default entries of the policy goldens as they are before this change (canonical SHA-256).
const DEFS = {
  probe: '01379858a38e07a02928ed54d080eb6e755fa201412128e42ba700efd81244e1',
  propose_files: '51ecbe87d8ce30999ed5b1c067069c72edf1ae6716c642d35489919f53f674ff',
  submit: '766e875db227c7922883c1981bde86dd9042bafb90c3cd8fa3c86636c5b9a0ff',
  submit_refinement: 'f7883c0926eb727694db419a51ca5e710cd6b2cc0079342a4a308fdeab389b89',
};
const POLICY_PROMPT = '50d0127915b768279c0d0fdf507cd1f82fb1ce828cea60f97b257e257242512f';
const POLICY_DEFAULT_TOOLS = 'b56e06bb74b301b319ba0189113e948e2b3ab4ad23fa444c9c540d546991b1f5';
const PROMPTS = '6632e079324c724df7437bdb12f4568d729bec8c490d7a81206c57a4a4edfc33';
const FORCE = {
  'force_ordinary_golden.json': '84441f9dce7504d84e467cce48e34ed4826d3e14560082ee26e4d8a7e29b11a8',
  'force_ordinary_models_golden.json': 'ec1bbe904491424083cc1f98497f3fa627615e58367689aab2810b6f66cdb43a',
};

// filesUnder are the repository-relative regular files under dir, sorted.
function filesUnder(dir) {
  const out = [];
  const walk = (d) => {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name);
      if (e.isDirectory()) walk(p);
      else if (e.isFile()) out.push(p);
    }
  };
  walk(dir);
  return out.sort();
}

if (require.main === module) {
  let line;
  try {
    const mode = process.argv[2];
    line = mode.startsWith('overlay-') ? overlayControl(mode) : control(mode);
  } catch (e) {
    line = `observer error ${e.message}`;
  }
  console.log(line);
  process.exitCode = line === 'observer accepted' ? 0 : 1;
} else {
  const { Given, Then } = require('@cucumber/cucumber');

  Given('kso: the pinned coordinator, synthesized to reject the shape_workers, shape_audit, shape_decision_check, review_material and max_rounds keys and to hide their fields', { timeout: 10 * 60 * 1000 }, function () {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'kso-synthetic-'));
    try {
      synthesize(process.cwd(), path.join(tmp, 'coordinator'));
    } catch (e) {
      fs.rmSync(tmp, { recursive: true, force: true });
      throw e;
    }
    this.ksoSynthetic = tmp;
  });

  Then('kso: go build, go vet and go test of every kogen-shaping package pass against it', { timeout: LIMIT + 5 * 60 * 1000 }, function () {
    try {
      const r = goChecks(process.cwd(), path.join(this.ksoSynthetic, 'coordinator'));
      if (!r.ok) throw new Error(`against the coordinator that rejects the removed keys, ${r.command}: ${r.status}\n${r.out}`);
      for (const pkg of ['github.com/KogenAI/kogen-shaping/shape', 'github.com/KogenAI/kogen-shaping/cmd/kogen-shape']) {
        assert.ok(new RegExp(`^ok\\s+${pkg.replace(/[.\/]/g, '\\$&')}\\s`, 'm').test(r.out), `${r.command}: ${r.status}, but it did not report ${pkg} ok:\n${r.out}`);
      }
    } finally {
      fs.rmSync(this.ksoSynthetic, { recursive: true, force: true });
    }
  });

  Then('kso: no file under shape or cmd names shape_workers, shape_audit, shape_decision_check, review_material or max_rounds', function () {
    const key = new RegExp(REMOVED_KEYS.join('|'));
    const named = [];
    for (const f of [...filesUnder('shape'), ...filesUnder('cmd')]) {
      fs.readFileSync(f, 'utf8').split('\n').forEach((l, i) => {
        if (key.test(l)) named.push(`${f}:${i + 1}: ${l.trim()}`);
      });
    }
    assert.deepEqual(named, [], `files that still name a removed key:\n${named.join('\n')}`);
  });

  Then('kso: a native Shape of a project that sets every opt-in offers only the default tools, answers shape_batch as an unknown tool, makes no audit or decision check request and publishes the submitted Intent unchanged', { timeout: 20 * 60 * 1000 }, function () {
    overlayTest('TestKShapeDropOptinsNative', {});
  });

  Then('kso: a claude-code shaper whose role sets max_rounds 1 under an executor shape cap of {int} rounds gets a cap of {int} rounds and publishes after 2 rounds', { timeout: 20 * 60 * 1000 }, function (cap, want) {
    overlayTest('TestKShapeDropOptinsExternal', { KDO_CASE: 'max_rounds', KDO_CAP: String(cap), KDO_WANT: String(want) });
  });

  Then('kso: a claude-code shaper of a project that sets every opt-in is not refused and publishes the submitted Intent from one session of one round', { timeout: 20 * 60 * 1000 }, function () {
    overlayTest('TestKShapeDropOptinsExternal', { KDO_CASE: 'opt-ins', KDO_CAP: '6', KDO_WANT: '6' });
  });

  Then('kso: the policy goldens drop only the shape_batch, worker_result and worker-role entries and keep every other entry unchanged', function () {
    const problems = [];
    const defs = golden('defs_golden.json');
    if (canonical(Object.keys(defs).sort()) !== canonical(Object.keys(DEFS))) problems.push(`defs_golden.json entries are ${Object.keys(defs).sort().join(', ')}, want ${Object.keys(DEFS).join(', ')}`);
    for (const [name, digest] of Object.entries(DEFS)) {
      if (name in defs && sha256(canonical(defs[name])) !== digest) problems.push(`defs_golden.json ${name} changed`);
    }
    const policy = golden('shaper_policy_golden.json');
    if (sha256(canonical(policy.prompt)) !== POLICY_PROMPT) problems.push('shaper_policy_golden.json prompt changed');
    const tools = Object.keys(policy.tools || {}).sort();
    if (canonical(tools) !== canonical(['default'])) problems.push(`shaper_policy_golden.json tool lists are ${tools.join(', ')}, want default`);
    if (sha256(canonical((policy.tools || {}).default)) !== POLICY_DEFAULT_TOOLS) problems.push('shaper_policy_golden.json default tools changed');
    if (sha256(canonical(golden('prompts_golden.json'))) !== PROMPTS) problems.push('prompts_golden.json changed');
    for (const [name, digest] of Object.entries(FORCE)) {
      if (sha256(fs.readFileSync(path.join('shape', 'testdata', name))) !== digest) problems.push(`${name} changed`);
    }
    assert.deepEqual(problems, [], problems.join('\n'));
  });
}
