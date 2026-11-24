'use strict';
const assert = require('node:assert/strict');
function refusal(result, code) {
  assert.equal(result.status, code, `process status; stderr: ${result.stderr}`);
  const doc = JSON.parse(result.stdout);
  assert.ok(doc && typeof doc === 'object' && !Array.isArray(doc), 'stdout must be one JSON object');
  assert.ok(result.stdout.endsWith('\n'), 'JSON result must end with a newline');
  assert.equal(doc.exit_code, code, 'JSON exit_code must match process status');
  assert.equal(doc.intent, null, 'a refusal must contain no Intent');
  assert.equal(typeof doc.error, 'string', 'error must be a diagnostic string');
  assert.ok(doc.error.length > 0, 'error must not be empty');
}
function fileSource(result, file) {
  const doc = JSON.parse(result.stdout);
  assert.ok(doc.error.includes(file), `error must name request file ${file}: ${doc.error}`);
}
function stdinSource(result) {
  const doc = JSON.parse(result.stdout);
  assert.match(doc.error, /\bstdin\b/i, 'error must identify stdin');
  assert.doesNotMatch(doc.error, /(?:^|[\s"'])(?:\/|[A-Za-z]:[\\/])\S*/, 'stdin error must not name a filesystem path');
}
module.exports = {refusal, fileSource, stdinSource};

if (require.main === module) {
  try {
    const file = '/fixture/request-source.bin';
    const fixture = error => ({status:2, stdout:JSON.stringify({exit_code:2, intent:null, error}) + '\n', stderr:''});
    const fromFile = fixture(`request ${file} is not valid UTF-8`);
    const fromStdin = fixture('request from stdin is not valid UTF-8');
    if (process.argv[2] === 'bad') {
      const doc = JSON.parse(fromStdin.stdout);
      doc.intent = {path:'/fixture/intent.md', content:'not a refusal'};
      fromStdin.stdout = JSON.stringify(doc) + '\n';
    }
    refusal(fromFile, 2);
    fileSource(fromFile, file);
    refusal(fromStdin, 2);
    stdinSource(fromStdin);
    console.log('accepted');
  } catch (_) {
    console.log('rejected');
    process.exitCode = 1;
  }
}
