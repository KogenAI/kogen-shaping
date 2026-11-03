'use strict';
// Kogen's control-run support hooks (Shape's unpassable-step check): they tell the preload which
// scenario and step are running, so a use of a stand-in is attributed to its step.
const {Before, After, BeforeStep, AfterStep} = require('@cucumber/cucumber');
const control = globalThis[Symbol.for('kogen.unpassable.control')];

Before(function (hook) { control.before(hook.pickle.id); });
BeforeStep(function (hook) { control.beforeStep(hook.pickle.id, hook.pickleStep.id); });
AfterStep(function (hook) { control.afterStep(hook.pickle.id, hook.pickleStep.id); });
After(function (hook) { control.after(hook.pickle.id); });
