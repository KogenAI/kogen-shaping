const {Then}=require('@cucumber/cucumber'); const assert=require('node:assert/strict');
Then('is-even: f of {word} is {string}', function(x, want) { const actual=require('../../f').f(x); assert.strictEqual(actual,want,'expected='+want+' observed='+actual); });
