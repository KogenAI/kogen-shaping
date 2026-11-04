'use strict';
// Kogen's preload of the wrong-implementation runs. Before any step file or implementation code
// loads, it freezes the module objects of node:assert and node:assert/strict, every
// function-valued property of each, and those functions' prototypes, so that no code can
// replace or disable an assertion by assigning to them.
for (const id of ['node:assert', 'node:assert/strict']) {
  const mod = require(id);
  for (const key of Reflect.ownKeys(mod)) {
    const desc = Object.getOwnPropertyDescriptor(mod, key);
    const value = desc && desc.value;
    if (typeof value === 'function') {
      if (value.prototype && typeof value.prototype === 'object') Object.freeze(value.prototype);
      Object.freeze(value);
    }
  }
  Object.freeze(mod);
}
