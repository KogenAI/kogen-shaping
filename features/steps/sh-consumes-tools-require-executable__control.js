'use strict';
// Control of SH-CONSUMES-TOOLS-REQUIRE-EXECUTABLE's harness, run in the checkout. It runs the two
// consumer tests through the harness, prints each test's outcome and the judgement of the
// "both tests pass" step, and exits 0 only when that judgement accepts. Cucumber loads this file
// with the other step files; it does nothing unless run as the main module.
//   good: the pinned kogen-verification, under which both tests pass before and after the Build.
//   bad:  the pinned one with the node-path text altered, under which TestToolsFromInstall fails.
const h = require('./sh-consumes-tools-require-executable__harness.js');

if (require.main === module) {
  const variant = {good: 'pinned', bad: 'pinned with the node-path text altered'}[process.argv[2]];
  if (!variant) {
    process.stderr.write('usage: sh-consumes-tools-require-executable__control.js good|bad\n');
    process.exitCode = 2;
  } else {
    const result = h.runConsumerTests(process.cwd(), variant);
    for (const test of h.TESTS) process.stdout.write(`${test}: ${result.outcomes[test] || 'did not run'}\n`);
    const accepted = h.passProblems(result).length === 0;
    process.stdout.write(accepted ? 'accepted\n' : 'rejected\n');
    process.exitCode = accepted ? 0 : 1;
  }
}
