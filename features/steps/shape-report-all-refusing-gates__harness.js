'use strict';
const fs = require('fs');
const os = require('os');
const path = require('path');
const cp = require('child_process');
const assert = require('assert/strict');
const root = process.cwd();
const driver = path.join(root, 'features/steps/shape-report-all-refusing-gates__driver/driver.go');
let directory, binary;
const cache = {};
function build() {
 directory = fs.mkdtempSync(path.join(os.tmpdir(), 'srag-'));
 const target = path.join(directory, 'driver_test.go');
 fs.writeFileSync(target, fs.readFileSync(driver, 'utf8').replace('//go:build ignore\n\n', ''));
 const overlay = path.join(directory, 'overlay.json');
 fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.join(root, 'shape/shape_report_all_refusing_gates_observation_test.go')]: target}}));
 binary = path.join(directory, 'observe');
 const r = cp.spawnSync('go', ['test', '-overlay', overlay, '-c', '-o', binary, './shape'], {cwd: root, encoding: 'utf8', timeout: 240000});
 assert.equal(r.status, 0, 'observation driver compilation failed: ' + r.stdout + r.stderr);
}
function observe(mode) {
 if (cache[mode]) return cache[mode];
 if (!binary) build();
 const output = path.join(directory, mode + '.json');
 const r = cp.spawnSync(binary, ['-test.run=^TestShapeReportAllRefusingGatesObservation$'], {cwd: path.join(root, 'shape'), env: {...process.env, SRAG_MODE: mode, SRAG_OUT: output}, encoding: 'utf8', timeout: 120000});
 assert.equal(r.status, 0, 'driver execution failed for ' + mode + ': ' + r.stdout + r.stderr);
 assert.ok(fs.existsSync(output), 'observation driver produced no result for ' + mode);
 return (cache[mode] = JSON.parse(fs.readFileSync(output, 'utf8')));
}
function cleanup() { if (directory) fs.rmSync(directory, {recursive: true, force: true}); directory = binary = undefined; }
module.exports = {observe, cleanup};
