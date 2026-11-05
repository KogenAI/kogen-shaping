'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const {spawnSync} = require('node:child_process');
const {When, Then} = require('@cucumber/cucumber');

const P = 'k-shape-fixture-title-user-name: ';
const OWN = 'k-shape-fixture-title-user-name__';

// sources lists the repository-relative Go files of the module and the JavaScript step files,
// without node_modules and without this Intent's own step files.
function sources() {
  const out = [];
  const walk = rel => {
    for (const e of fs.readdirSync(rel || '.', {withFileTypes: true})) {
      const p = rel ? rel + '/' + e.name : e.name;
      if (e.isDirectory()) {
        if (e.name !== 'node_modules' && e.name !== '.git') walk(p);
      } else if (e.isFile() && (p.endsWith('.go') || (p.startsWith('features/steps/') && p.endsWith('.js'))) && !e.name.startsWith(OWN)) {
        out.push(p);
      }
    }
  };
  walk('');
  return out.sort();
}

function all(text, re) {
  return [...text.matchAll(re)].map(m => m[1].trim());
}

// A Git user name configured for a fixture repository, and the literal title of an Intent
// built as "# <title>\n\nSlug:" (or "# <title>" + slug + "\n\nSlug:").
const nameRes = [/"config",\s*"user\.name",\s*"([^"]+)"/g, /"user\.name=([^"]+)"/g];
const titleRe = /"# ([^"\\]+?)(?:"\s*\+\s*[A-Za-z_]\w*\s*\+\s*")?\\n\\nSlug:/g;

// namesWord reports whether title contains name as a whole word, ignoring case.
function namesWord(title, name) {
  const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return new RegExp('(?<![\\p{L}\\p{N}\\p{M}_])' + escaped + '(?![\\p{L}\\p{N}\\p{M}_])', 'iu').test(title);
}

When(P + 'the module\'s fixtures are scanned for Git user names and Intent titles', function () {
  this.fixtures = sources().flatMap(file => {
    const text = fs.readFileSync(file, 'utf8');
    const names = nameRes.flatMap(re => all(text, re));
    const titles = all(text, titleRe);
    return names.length && titles.length ? [{file, names, titles}] : [];
  });
});

Then(P + 'the fixtures that configure a Git user name and build an Intent title are exactly these files:', function (table) {
  const want = table.raw().map(r => r[0]);
  assert.deepStrictEqual(this.fixtures.map(f => f.file), want, 'fixtures found (file, Git user names, Intent titles): ' + JSON.stringify(this.fixtures));
});

Then(P + 'no such fixture titles its Intent with its Git user name', function () {
  const clashes = [];
  for (const f of this.fixtures) {
    for (const title of f.titles) for (const name of f.names) if (namesWord(title, name)) clashes.push(`${f.file}: Intent title "${title}" contains Git user name "${name}"`);
  }
  assert.equal(clashes.length, 0, clashes.join('\n') + '\nfixtures: ' + JSON.stringify(this.fixtures));
});

When(P + 'the module\'s Go files and step files are searched for coordinator title-lint wording', function () {
  this.texts = sources().map(file => ({file, text: fs.readFileSync(file, 'utf8')}));
});

Then(P + 'none of them contains any of:', function (table) {
  const phrases = table.raw().map(r => r[0]);
  assert.ok(this.texts.some(t => t.file === 'shape/approve_test.go'), 'the search did not reach shape/approve_test.go');
  const hits = [];
  for (const {file, text} of this.texts) for (const phrase of phrases) if (text.includes(phrase)) hits.push(`${file}: "${phrase}"`);
  assert.equal(hits.length, 0, 'coordinator title-lint wording found:\n' + hits.join('\n'));
});

When(P + 'go test runs {string} in {string}', {timeout: 300000}, function (filter, pkg) {
  const env = {...process.env, GOWORK: 'off'};
  for (const k of Object.keys(env)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete env[k];
  const args = ['test', '-count=1', '-json', '-run', filter, pkg];
  const r = spawnSync('go', args, {encoding: 'utf8', env, timeout: 280000, maxBuffer: 64 * 1024 * 1024});
  this.goTest = {...r, command: 'go ' + args.join(' ')};
});

Then(P + 'each of these tests passes exactly once:', function (table) {
  const r = this.goTest;
  const text = (r.stdout || '') + (r.stderr || '');
  const events = text.split('\n').flatMap(line => { try { return [JSON.parse(line)]; } catch { return []; } });
  for (const [name] of table.raw()) {
    const actions = events.filter(e => e.Test === name && ['pass', 'fail', 'skip'].includes(e.Action)).map(e => e.Action);
    const outcome = actions.length ? actions.join(',') : 'absent (ran zero times)';
    assert.deepStrictEqual(actions, ['pass'], `required test ${name}: ${outcome}; command ${r.command}; exit ${r.status} ${r.error || ''}\n${text.slice(-4000)}`);
  }
  assert.equal(r.status, 0, `required tests ${table.raw().map(x => x[0]).join(', ')}; command ${r.command}; exit ${r.status} ${r.error || ''}\n${text.slice(-4000)}`);
});

Then(P + 'go.mod requires kogen-coordinator no older than {string}', function (version) {
  const stamp = v => {
    const m = /^v\d+\.\d+\.\d+-(?:0\.)?(\d{14})-[0-9a-f]{12}$/.exec(v);
    assert.ok(m, `not a pseudo-version: ${v}`);
    return m[1];
  };
  const m = /^\s*github\.com\/KogenAI\/kogen-coordinator\s+(\S+)/m.exec(fs.readFileSync('go.mod', 'utf8'));
  assert.ok(m, 'go.mod does not require github.com/KogenAI/kogen-coordinator');
  assert.ok(stamp(m[1]) >= stamp(version), `go.mod requires kogen-coordinator ${m[1]}, older than ${version}`);
});
