'use strict';
// Observer control for K-SHAPING-LOOP-MODULE-PATH:
//   "$KOGEN_NODE" features/steps/k-shaping-loop-module-path__control.js good|bad
// It copies the current checkout (without .git, node_modules and .kogen/local) into a temporary
// directory, converts that copy to the github.com/KogenAI/kogen-loop module path at the pinned
// versions (go.mod, go.sum, imports and the two kogen-loop tool descriptions in the Shaper policy
// golden), and runs every check of the checks module on it. "bad" then plants three defects that only deeper checks see: an unused
// replace directive, a features/steps .go.txt harness with a bare import and a changed policy golden.
// It prints one line per check and exits 0 when no check found a problem, 1 when one did and 3
// when the fixture could not be prepared. It never changes the checkout.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const c = require('./k-shaping-loop-module-path__checks.js');

const LOOP = 'github.com/KogenAI/kogen-loop';
const PINS = [
  {module: 'github.com/KogenAI/kogen-coordinator', version: 'v0.0.0-20261105022408-e7bc7ce9c169'},
  {module: 'github.com/KogenAI/kogen-executor', version: 'v0.0.0-20261105010505-8d465842f9df'},
  {module: LOOP, version: 'v0.0.0-20261105005409-31036b2c02e8'},
  {module: 'github.com/KogenAI/kogen-verification', version: 'v0.0.0-20261105003813-ac4f6a9a4877'},
];
const GOSUM = `github.com/KogenAI/kogen-coordinator v0.0.0-20261105022408-e7bc7ce9c169 h1:1JQTfBtT0MT1ad13QTDBxuGqSZRaekRHR6qNjQYr9NU=
github.com/KogenAI/kogen-coordinator v0.0.0-20261105022408-e7bc7ce9c169/go.mod h1:MMc15p3YwlY71dMxfCdX6QlR2jXJbSxL6YA9Y7971c8=
github.com/KogenAI/kogen-executor v0.0.0-20261105010505-8d465842f9df h1:ZyDgxjdzyMFmfxahK+vs08TnUTwgf9wiasGtft9K3/o=
github.com/KogenAI/kogen-executor v0.0.0-20261105010505-8d465842f9df/go.mod h1:qjm/u2kVwQdMxIPTbNdExSqanEcPs8UpBUFldsotEDg=
github.com/KogenAI/kogen-loop v0.0.0-20261105005409-31036b2c02e8 h1:FHmM4ohzS7si6JRIsOg/9mxVIV03ceLAwKFjXboAZPU=
github.com/KogenAI/kogen-loop v0.0.0-20261105005409-31036b2c02e8/go.mod h1:yPduvTAVdxH5egqW82Uis/ZhemRb5gNSdzTll/f6bG0=
github.com/KogenAI/kogen-verification v0.0.0-20261105003813-ac4f6a9a4877 h1:CP621ikt9tS0kUwL/16WPFGXoCDjDD4ogKRWYZ4fVjw=
github.com/KogenAI/kogen-verification v0.0.0-20261105003813-ac4f6a9a4877/go.mod h1:a434oUbU/VdUlUld0N+kUL4k7ZaEdVummnkHaPyDCiU=
`;
const POLICY = 'shape/testdata/shaper_policy_golden.json';
const GOLDENS = [
  [POLICY, '98c842c3fb874fc529e08dd3d0c1617575c974a1965c0924b3769061aa7e1e9c'],
  ['shape/testdata/prompts_golden.json', '7b34f311c75d8c82f4afc0749c8fb36e6afd69d3f85473970e79fc27e023df41'],
  ['shape/testdata/defs_golden.json', 'c03bf24a45e5ffd4a1e5de8b38f6b8112a76528c35f74c21fba185ad41cbb8ec'],
];
// The descriptions the pinned kogen-loop gives list_files and search_text.
const DESCRIPTIONS = {
  list_files: "List the repository's files (tracked and untracked, not ignored), optionally under a folder inside the repository root; paths outside it, symlinks leaving it and ~ are refused. Each path is printed as named, except that a name holding a control or other non-printable character or invalid UTF-8, or starting with \", is printed as a Go double-quoted string literal (for example \"a\\nb.txt\"): decode it to get the path.",
  search_text: "Search the repository's files for a regular expression, optionally under a path inside the repository root; paths outside it, symlinks leaving it and ~ are refused. Patterns use RE2 syntax (Go's regexp package) on every host and match within one line: \\b, \\d, \\s, \\w, alternation and classes work, and a pattern RE2 cannot express, such as a backreference or lookaround, returns an error instead of no matches. Returns path:line:text matches. Each path is printed as named, except that a name holding a control or other non-printable character or invalid UTF-8, or starting with \", is printed as a Go double-quoted string literal (for example \"a\\nb.txt\"): decode it to get the path.",
};
const PLANTED = 'features/steps/k_planted_bare_import_test.go.txt';

function fail(message) {
  throw new Error(message);
}

function rewrite(file, from, to) {
  const text = fs.readFileSync(file, 'utf8');
  if (text.includes(from)) fs.writeFileSync(file, text.split(from).join(to));
}

function goFiles(dir, out) {
  for (const e of fs.readdirSync(dir, {withFileTypes: true})) {
    if (e.name === '.git' || e.name === 'node_modules') continue;
    const p = path.join(dir, e.name);
    if (e.isDirectory()) goFiles(p, out);
    else if (e.name.endsWith('.go')) out.push(p);
  }
  return out;
}

function main(mode) {
  if (mode !== 'good' && mode !== 'bad') {
    console.log('control could not be prepared: usage: control.js good|bad');
    process.exitCode = 3;
    return;
  }
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'k-shaping-loop-module-path-control-'));
  let failed = false;
  try {
    const root = path.join(tmp, 'checkout');
    fs.mkdirSync(root);
    const skip = new Set(['.git', 'node_modules', path.join('.kogen', 'local')]);
    fs.cpSync(process.cwd(), root, {recursive: true, verbatimSymlinks: true, filter: src => {
      const rel = path.relative(process.cwd(), src);
      return !skip.has(rel) && !skip.has(path.basename(rel));
    }});
    if (!fs.existsSync(path.join(root, 'go.mod'))) fail(`${process.cwd()} holds no go.mod`);
    const env = {...process.env, GOFLAGS: '', GOWORK: ''};
    const original = fs.readFileSync(path.join(root, POLICY), 'utf8');

    // The converted module.
    c.go(root, ['mod', 'edit', '-droprequire=kogen-loop', '-dropreplace=kogen-loop', ...PINS.map(p => `-require=${p.module}@${p.version}`)], env);
    fs.writeFileSync(path.join(root, 'go.sum'), GOSUM);
    // This Intent's own step files sit in features/steps as well, before and after the Build.
    for (const name of ['checks', 'control', 'steps']) {
      const file = `k-shaping-loop-module-path__${name}.js`;
      fs.copyFileSync(path.join(__dirname, file), path.join(root, 'features/steps', file));
    }
    const to = String.fromCharCode(34) + LOOP + '/';
    for (const f of goFiles(root, [])) rewrite(f, c.BARE, to);
    for (const name of fs.readdirSync(path.join(root, 'features/steps'))) {
      if (name.endsWith('.js') || name.endsWith('.go.txt')) rewrite(path.join(root, 'features/steps', name), c.BARE, to);
    }
    let policy = original;
    for (const tool of JSON.parse(original).tools.default) {
      if (DESCRIPTIONS[tool.name]) policy = policy.replace(JSON.stringify(tool.description), JSON.stringify(DESCRIPTIONS[tool.name]));
    }
    fs.writeFileSync(path.join(root, POLICY), policy);

    if (mode === 'bad') {
      c.go(root, ['mod', 'edit', '-replace=example.com/unused=example.com/unused@v1.0.0'], env);
      fs.writeFileSync(path.join(root, PLANTED), 'package shape\n\nimport (\n\t' + c.BARE + 'sys"\n)\n');
      fs.writeFileSync(path.join(root, POLICY), policy + '\n');
    }

    const report = (name, problems) => {
      if (problems.length) failed = true;
      console.log(`${name}: ${problems.length ? problems.join(' | ') : 'ok'}`);
    };
    const mod = c.goMod(root, env);
    report('require', c.requireProblems(mod, PINS));
    report('replace', c.replaceProblems(mod));
    report('go.sum', c.goSumProblems(root, PINS));
    report('go.work', c.goWorkProblems(root, env));
    const build = c.listBuild(root, env);
    report('bare paths', c.barePathProblems(build, 'kogen-loop', 'kogen-loop/'));
    report('build list', c.buildListProblems(build, `${LOOP} v0.0.0-20261105005409-31036b2c02e8`));
    report('imports', c.importProblems(build, ['provider', 'sys', 'tools'].map(p => `${LOOP}/${p}`)));
    report('harnesses', c.harnessProblems(root, 'features/steps', c.BARE));
    // The request's grep for bare imports in Go files, assembled from c.BARE.
    const grep = c.shell(root, 'grep -rn "\\' + c.BARE + '" --include=*.go .', 60 * 1000);
    report('grep', grep.status === 1 && grep.output === '' ? [] : [`grep exited ${grep.status} with ${grep.output.split('\n').filter(Boolean).length} lines`]);
    for (const [rel, sha] of GOLDENS) report(rel, c.shaProblems(root, rel, sha));
    process.exitCode = failed ? 1 : 0;
  } catch (e) {
    console.log(`control could not be prepared: ${e.message}`);
    process.exitCode = 3;
  } finally {
    fs.rmSync(tmp, {recursive: true, force: true});
  }
}

// Acceptance loads every features/steps/*.js file; the control runs only as a program.
if (require.main === module) main(process.argv[2]);
