'use strict';
const assert = require('node:assert/strict');
const { Then } = require('@cucumber/cucumber');
const { runCase } = require('./shape-batch-independent-tool-calls__driver/run.js');

const slow = { timeout: 600000 };

// check runs one injected Go case and turns its failure into an assertion of this step file.
function check(name, env) {
  try {
    runCase(name, env);
  } catch (e) {
    assert.fail(e.message);
  }
}

Then('shape-batch-independent-tool-calls: the overlap case passes', slow, function () {
  check('overlap', {});
});

Then('shape-batch-independent-tool-calls: the barriers case passes', slow, function () {
  check('barriers', {});
});

Then('shape-batch-independent-tool-calls: a repeated unchanged read of README.md answers {string}', slow, function (note) {
  check('repeat', { ACCEPT_BATCH_README_NOTE: note });
});

Then('shape-batch-independent-tool-calls: shaper.md, the shaper instructions and the read_file, search_text, list_files and probe descriptions contain {string}', slow, function (sentence) {
  check('policy', { ACCEPT_BATCH_SENTENCE: sentence });
});
