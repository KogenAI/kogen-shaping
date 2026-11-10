const {Then}=require('@cucumber/cucumber');
const {execFileSync}=require('node:child_process');
const assert=require('node:assert/strict');
Then('shape-bounds-tool-free-answers: {word} obeys the bound',function(name){
 try {
 execFileSync(process.execPath,[require.resolve('./shape-bounds-tool-free-answers__driver/run.js'),name],{stdio:'pipe',timeout:180000});
 } catch(e) { assert.fail(`${name}: ${e.stderr || e.message}`); }
});
