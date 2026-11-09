'use strict';
// Kogen's control-run preload (Shape's unpassable-step check). It loads before any step file
// (node -r) and replaces everything acceptance code obtains from outside itself with a
// stand-in: a value on which every operation by acceptance code is a use and fails the step.
// Acceptance code means the JavaScript files under the configured steps folder. Operations by any other
// code (Node's assert internals, Cucumber's formatters) are inert and never count, so a
// stand-in can not break Cucumber's own reporting.
const Module = require('node:module');
const fs = require('node:fs');
const path = require('node:path');
const url = require('node:url');

const KEY = Symbol.for('kogen.unpassable.control');
const logPath = process.env.KOGEN_UNPASSABLE_LOG;
const stepsDir = process.env.KOGEN_UNPASSABLE_STEPS_DIR;
delete process.env.KOGEN_UNPASSABLE_LOG;
delete process.env.KOGEN_UNPASSABLE_STEPS_DIR;
const appendFileSync = fs.appendFileSync;
const cwd = process.cwd();
let realCwd = cwd;
try { realCwd = fs.realpathSync(cwd); } catch (e) { /* the working directory is used as it is */ }
const stepsDirs = [...new Set([cwd, realCwd])].map(d => path.resolve(d, stepsDir) + path.sep);
const self = __filename;

function toPath(file) {
  return typeof file === 'string' && file.startsWith('file://') ? url.fileURLToPath(file) : file;
}
function underSteps(file) {
  file = toPath(file);
  return typeof file === 'string' && stepsDirs.some(d => file.startsWith(d));
}
function isAcceptanceFile(file) {
  file = toPath(file);
  return underSteps(file) && /\.(c|m)?js$/.test(file);
}

function frames() {
  const prepare = Error.prepareStackTrace, limit = Error.stackTraceLimit;
  try {
    const holder = {};
    Error.stackTraceLimit = 64;
    Error.prepareStackTrace = (_, sites) => sites;
    Error.captureStackTrace(holder);
    return Array.isArray(holder.stack) ? holder.stack : [];
  } finally {
    Error.prepareStackTrace = prepare;
    Error.stackTraceLimit = limit;
  }
}

// byAcceptance reports whether the code that called the function this one is called from is
// acceptance code: its first frame that is neither native, this preload nor util's promisify.
function byAcceptance() {
  for (const site of frames()) {
    const file = site.getFileName();
    if (!file || file === self || file === 'node:internal/util') continue;
    return isAcceptanceFile(file);
  }
  return false;
}

const state = {pickle: null, step: null};
function log(record) {
  if (!logPath) return;
  try { appendFileSync(logPath, JSON.stringify(record) + '\n'); } catch (e) { /* the control then has no evidence */ }
}
// use records a use inside a scenario and reports true; outside a scenario (while the step
// files load) nothing is recorded and false is reported.
function use(what) {
  if (state.pickle === null) return false;
  log({k: 'use', pickle: state.pickle, step: state.step, what: String(what)});
  return true;
}

function standIn(label) {
  const target = function () {};
  const operation = (op, outside, inert) => {
    if (!byAcceptance()) return inert();
    if (use(label + ' ' + op)) throw new Error('Kogen control run: ' + label + ' is outside acceptance code and was used (' + op + ')');
    return outside();
  };
  return new Proxy(target, {
    get: (t, key, receiver) => operation('get ' + String(key),
      () => typeof key === 'symbol' ? Reflect.get(t, key, receiver) : standIn(label + '.' + key), () => Reflect.get(t, key, receiver)),
    set: (t, key, value, receiver) => operation('set ' + String(key), () => Reflect.set(t, key, value, receiver), () => Reflect.set(t, key, value, receiver)),
    has: (t, key) => operation('in ' + String(key), () => Reflect.has(t, key), () => Reflect.has(t, key)),
    deleteProperty: (t, key) => operation('delete ' + String(key), () => Reflect.deleteProperty(t, key), () => Reflect.deleteProperty(t, key)),
    defineProperty: (t, key, descriptor) => operation('define ' + String(key), () => Reflect.defineProperty(t, key, descriptor), () => Reflect.defineProperty(t, key, descriptor)),
    getOwnPropertyDescriptor: (t, key) => operation('describe ' + String(key), () => Reflect.getOwnPropertyDescriptor(t, key), () => Reflect.getOwnPropertyDescriptor(t, key)),
    ownKeys: t => operation('keys', () => Reflect.ownKeys(t), () => Reflect.ownKeys(t)),
    getPrototypeOf: t => operation('prototype', () => Reflect.getPrototypeOf(t), () => Reflect.getPrototypeOf(t)),
    setPrototypeOf: (t, proto) => operation('set prototype', () => Reflect.setPrototypeOf(t, proto), () => Reflect.setPrototypeOf(t, proto)),
    apply: () => operation('call', () => standIn(label + '()'), () => undefined),
    construct: () => operation('construct', () => standIn('new ' + label), () => ({})),
  });
}

// The hooks Kogen's support file registers with Cucumber: where a scenario and its steps are.
globalThis[KEY] = {
  standIn,
  before(pickle) { state.pickle = pickle; state.step = null; log({k: 'before', pickle}); },
  beforeStep(pickle, step) { state.pickle = pickle; state.step = step; log({k: 'beforeStep', pickle, step}); },
  afterStep(pickle, step) { log({k: 'afterStep', pickle, step}); state.step = null; },
  after(pickle) { log({k: 'after', pickle}); state.pickle = null; state.step = null; },
};

// Modules loaded by acceptance code: Node builtins, @cucumber/cucumber and files under
// the configured steps folder stay real; any other module, resolvable or not, is a stand-in.
function isCucumber(request) {
  return request === '@cucumber/cucumber' || request.startsWith('@cucumber/cucumber/');
}
function isBuiltin(request) {
  return request.startsWith('node:') || Module.builtinModules.includes(request);
}
const load = Module._load;
Module._load = function (request, parent, isMain) {
  if (parent && typeof request === 'string' && isAcceptanceFile(parent.filename) && !isBuiltin(request) && !isCucumber(request)) {
    let inside = false;
    try { inside = underSteps(Module._resolveFilename(request, parent, false)); } catch (e) { /* unresolvable */ }
    if (!inside) return standIn('module ' + request);
  }
  return load.apply(this, arguments);
};
if (typeof Module.registerHooks === 'function') {
  const standInURL = 'file:///kogen-standin/';
  Module.registerHooks({
    resolve(specifier, context, next) {
      if (!context.parentURL || !isAcceptanceFile(context.parentURL) || isBuiltin(specifier) || isCucumber(specifier)) return next(specifier, context);
      try {
        const resolved = next(specifier, context);
        if (resolved.url.startsWith('node:') || underSteps(resolved.url)) return resolved;
      } catch (e) { /* unresolvable */ }
      return {url: standInURL + encodeURIComponent(specifier) + '.mjs', shortCircuit: true};
    },
    load(location, context, next) {
      if (!location.startsWith(standInURL)) return next(location, context);
      const request = decodeURIComponent(location.slice(standInURL.length, -'.mjs'.length));
      // Only the default export is a stand-in: a named import of it does not link, and the step file does not load.
      return {format: 'module', shortCircuit: true,
        source: 'export default globalThis[Symbol.for("kogen.unpassable.control")].standIn(' + JSON.stringify('module ' + request) + ');'};
    },
  });
}

// Wrappers of the functions whose result is a stand-in for acceptance code (kind "result"), or
// whose result can not be one and whose call is a use (kind "use"). Calls by other code reach
// the original unchanged. The wrappers drop util.promisify.custom, so a promisified call ends
// up here too.
function wrap(owner, name, kind) {
  const original = owner && owner[name];
  if (typeof original !== 'function') return;
  const label = (owner === fs ? 'fs.' : '') + name;
  const wrapper = function (...args) {
    if (byAcceptance()) {
      if (kind === 'result') return standIn(label + '()');
      use(label + '()');
    }
    return original.apply(this, args);
  };
  for (const key of Object.getOwnPropertyNames(original)) {
    if (!['length', 'name', 'prototype', 'arguments', 'caller'].includes(key)) wrapper[key] = original[key];
  }
  if (typeof original.native === 'function') wrap(original, 'native', kind), wrapper.native = original.native;
  owner[name] = wrapper;
}

const reads = ['readFileSync', 'readdirSync', 'statSync', 'lstatSync', 'readlinkSync', 'realpathSync', 'opendirSync', 'openSync', 'globSync'];
const probes = ['existsSync', 'accessSync'];
const asynchronous = ['readFile', 'readdir', 'stat', 'lstat', 'access', 'exists', 'readlink', 'realpath', 'opendir', 'open', 'createReadStream', 'glob'];
for (const name of reads) wrap(fs, name, 'result');
for (const name of probes) wrap(fs, name, 'use');
for (const name of asynchronous) wrap(fs, name, 'use');
for (const name of asynchronous) wrap(fs.promises, name, 'use');

const cp = require('node:child_process');
for (const name of ['spawn', 'spawnSync', 'exec', 'execSync', 'execFile', 'execFileSync', 'fork']) wrap(cp, name, 'result');
const net = require('node:net'), tls = require('node:tls'), http = require('node:http'), https = require('node:https'), dgram = require('node:dgram');
for (const name of ['connect', 'createConnection']) wrap(net, name, 'result');
wrap(tls, 'connect', 'result');
for (const name of ['request', 'get']) { wrap(http, name, 'result'); wrap(https, name, 'result'); }
wrap(dgram, 'createSocket', 'result');
wrap(globalThis, 'fetch', 'result');

// Reading process.env from acceptance code is a use.
let env = process.env;
Object.defineProperty(process, 'env', {
  configurable: true,
  enumerable: true,
  get() {
    if (byAcceptance()) use('process.env');
    return env;
  },
  set(value) { env = value; },
});
