'use strict';
// Runs the protected Go journey TestKShapeBlockedReportRepairAcceptance against this module's
// real Shape command with a scripted shaper. The Go source is written to a temporary file and
// compiled into the shape test binary with a go -overlay; it is never written into the checkout.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const TEST = 'TestKShapeBlockedReportRepairAcceptance';
const GO = String.raw`package shape

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const kbrNoRetain = "\nNo proposals from this submission were retained; earlier proposals are unchanged."

const kbrEditable = " is not protected by the current contract: the builder may edit it, so it is no reason to block"

const kbrNone = "none of the reported paths is protected: withdraw the blocked report and submit the Intent files"

const kbrSome = "keep only the protected paths in the blocked report, with its reason, or withdraw the report and submit the Intent files if no protected edit is required"

const kbrNext = "Next: revise the request, or review and commit .kogen/project.yaml outside Build, then shape again; preserve checks and actual check configuration. There is no protected-edit override.\n"

const kbrBoth = "[\".kogen/project.yaml\", \"config/deploy.cfg\"]"

func kbrSet(list string) string {
	return "protected paths under the current contract (configured protect patterns and automatic baseline): " + list
}

// kbrObs is what one journey observed; kbrJudge decides it, so the controls judge the same way.
type kbrObs struct {
	outputs   map[string]string
	exit      int
	err       string
	stdout    string
	requests  int
	published bool
	written   bool
	headMoved bool
	status    string
	cfg       string
}

func kbrWant(o kbrObs, id, want string) []string {
	if got := o.outputs[id]; got != want {
		return []string{fmt.Sprintf("tool output %s = %q, want %q", id, got, want)}
	}
	return nil
}

func kbrUntouched(o kbrObs) []string {
	var p []string
	if o.published || o.written {
		p = append(p, "a blocked report wrote an Intent, an approval or a run")
	}
	if o.headMoved || o.status != "" {
		p = append(p, fmt.Sprintf("a blocked report changed the repository: HEAD moved %v, git status %q", o.headMoved, o.status))
	}
	if o.cfg != "strict=true\n" {
		p = append(p, fmt.Sprintf("protected config/deploy.cfg = %q", o.cfg))
	}
	return p
}

func kbrJudge(mode string, o kbrObs) []string {
	var p []string
	result := func(exit int) {
		if o.exit != exit || o.err != "" {
			p = append(p, fmt.Sprintf("shape exit %d, error %q, want exit %d and no error; stdout:\n%s", o.exit, o.err, exit, o.stdout))
		}
	}
	switch mode {
	case "unprotected":
		p = append(p, kbrWant(o, "b1", "invalid submission:\n"+
			"blocked path deploy/service.yaml"+kbrEditable+"\n"+
			"blocked path calc.js"+kbrEditable+"\n"+
			"blocked path README.md"+kbrEditable+"\n"+
			kbrSet(kbrBoth)+"\n"+kbrNone+kbrNoRetain)...)
		result(0)
		if !o.published {
			p = append(p, "the files submitted after the refusal were not published")
		}
	case "mixed":
		p = append(p, kbrWant(o, "b1", "invalid submission:\n"+
			"blocked path calc.js"+kbrEditable+"\n"+
			"blocked path README.md"+kbrEditable+"\n"+
			kbrSet(kbrBoth)+"\n"+kbrSome+kbrNoRetain)...)
		result(1)
		if want := "shape/protected_conflict: required edits touch protected files: config/deploy.cfg\nthe deployment check definition must change\n" + kbrNext; o.stdout != want {
			p = append(p, fmt.Sprintf("stdout = %q, want %q", o.stdout, want))
		}
		p = append(p, kbrUntouched(o)...)
	case "protected":
		result(1)
		if want := "shape/protected_conflict: required edits touch protected files: config/deploy.cfg, .kogen/project.yaml\nthe deployment check and its configuration must change\n" + kbrNext; o.stdout != want {
			p = append(p, fmt.Sprintf("stdout = %q, want %q", o.stdout, want))
		}
		if o.requests != 1 {
			p = append(p, fmt.Sprintf("requests = %d, want 1: the protected report was accepted at once", o.requests))
		}
		p = append(p, kbrUntouched(o)...)
	case "malformed":
		p = append(p, kbrWant(o, "b1", "invalid submission:\nblocked.paths must be a non-empty list\nblocked.reason must be non-empty text"+kbrNoRetain)...)
		p = append(p, kbrWant(o, "b2", "invalid submission:\nblocked.paths must contain non-empty paths"+kbrNoRetain)...)
		result(5)
	case "automatic":
		p = append(p, kbrWant(o, "b1", "invalid submission:\n"+
			"blocked path calc.js"+kbrEditable+"\n"+
			kbrSet("[\".kogen/project.yaml\"]")+"\n"+kbrNone+kbrNoRetain)...)
		result(0)
		if !o.published {
			p = append(p, "the files submitted after the refusal were not published")
		}
	default:
		p = append(p, "unknown mode "+mode)
	}
	return p
}

// kbrGood are the observations of a correct implementation, one per mode.
func kbrGood() map[string]kbrObs {
	untouched := kbrObs{cfg: "strict=true\n"}
	g := map[string]kbrObs{}
	o := untouched
	o.outputs = map[string]string{"b1": "invalid submission:\nblocked path deploy/service.yaml" + kbrEditable + "\nblocked path calc.js" + kbrEditable + "\nblocked path README.md" + kbrEditable + "\n" + kbrSet(kbrBoth) + "\n" + kbrNone + kbrNoRetain}
	o.published, o.written, o.requests = true, true, 2
	g["unprotected"] = o
	o = untouched
	o.outputs = map[string]string{"b1": "invalid submission:\nblocked path calc.js" + kbrEditable + "\nblocked path README.md" + kbrEditable + "\n" + kbrSet(kbrBoth) + "\n" + kbrSome + kbrNoRetain}
	o.exit, o.requests = 1, 2
	o.stdout = "shape/protected_conflict: required edits touch protected files: config/deploy.cfg\nthe deployment check definition must change\n" + kbrNext
	g["mixed"] = o
	o = untouched
	o.exit, o.requests = 1, 1
	o.stdout = "shape/protected_conflict: required edits touch protected files: config/deploy.cfg, .kogen/project.yaml\nthe deployment check and its configuration must change\n" + kbrNext
	g["protected"] = o
	o = untouched
	o.outputs = map[string]string{"b1": "invalid submission:\nblocked.paths must be a non-empty list\nblocked.reason must be non-empty text" + kbrNoRetain, "b2": "invalid submission:\nblocked.paths must contain non-empty paths" + kbrNoRetain}
	o.exit, o.requests = 5, 3
	g["malformed"] = o
	o = untouched
	o.outputs = map[string]string{"b1": "invalid submission:\nblocked path calc.js" + kbrEditable + "\n" + kbrSet("[\".kogen/project.yaml\"]") + "\n" + kbrNone + kbrNoRetain}
	o.published, o.written, o.requests = true, true, 2
	g["automatic"] = o
	return g
}

// kbrBad are known-bad observations, each breaking one assertion of a mode.
func kbrBad() []struct {
	mode, name string
	o          kbrObs
} {
	g := kbrGood()
	var out []struct {
		mode, name string
		o          kbrObs
	}
	add := func(mode, name string, change func(o *kbrObs)) {
		o := g[mode]
		outputs := map[string]string{}
		for k, v := range o.outputs {
			outputs[k] = v
		}
		o.outputs = outputs
		change(&o)
		out = append(out, struct {
			mode, name string
			o          kbrObs
		}{mode, name, o})
	}
	add("unprotected", "the unchanged one-line refusal", func(o *kbrObs) {
		o.outputs["b1"] = "invalid submission:\nblocked path deploy/service.yaml is not protected by the current contract\nblocked path calc.js is not protected by the current contract\nblocked path README.md is not protected by the current contract" + kbrNoRetain
	})
	add("unprotected", "files refused after the refusal", func(o *kbrObs) { o.exit, o.published = 1, false })
	add("unprotected", "an empty protected set", func(o *kbrObs) { o.outputs["b1"] = strings.Replace(o.outputs["b1"], kbrBoth, "[]", 1) })
	add("mixed", "the protected path named as unprotected", func(o *kbrObs) {
		o.outputs["b1"] = strings.Replace(o.outputs["b1"], "\nblocked path README.md", "\nblocked path config/deploy.cfg"+kbrEditable+"\nblocked path README.md", 1)
	})
	add("mixed", "the next move for unprotected-only reports", func(o *kbrObs) { o.outputs["b1"] = strings.Replace(o.outputs["b1"], kbrSome, kbrNone, 1) })
	add("mixed", "the corrected report refused", func(o *kbrObs) { o.exit, o.stdout = 5, "" })
	add("mixed", "a blocked report that published", func(o *kbrObs) { o.published = true })
	add("protected", "a refused protected report", func(o *kbrObs) { o.requests = 2 })
	add("protected", "a protected file edited", func(o *kbrObs) { o.cfg, o.status = "strict=false\n", " M config/deploy.cfg\n" })
	add("protected", "a commit", func(o *kbrObs) { o.headMoved = true })
	add("protected", "a different stdout", func(o *kbrObs) { o.stdout = strings.Replace(o.stdout, ", .kogen/project.yaml", "", 1) })
	add("malformed", "extra lines for empty paths", func(o *kbrObs) {
		o.outputs["b1"] = strings.Replace(o.outputs["b1"], kbrNoRetain, "\n"+kbrSet(kbrBoth)+kbrNoRetain, 1)
	})
	add("malformed", "extra lines for a blank path", func(o *kbrObs) {
		o.outputs["b2"] = strings.Replace(o.outputs["b2"], kbrNoRetain, "\n"+kbrNone+kbrNoRetain, 1)
	})
	add("malformed", "a wrong exit", func(o *kbrObs) { o.exit = 1 })
	add("automatic", "an empty automatic set", func(o *kbrObs) {
		o.outputs["b1"] = strings.Replace(o.outputs["b1"], "[\".kogen/project.yaml\"]", "[]", 1)
	})
	add("automatic", "files not published", func(o *kbrObs) { o.published = false })
	return out
}

func kbrBlocked(id string, paths []any, reason any) shpStep {
	return shpSubmit(id, map[string]any{"blocked": map[string]any{"paths": paths, "reason": reason}})
}

func kbrProtect(f *shpFx) {
	f.write("config/deploy.cfg", "strict=true\n")
	f.write("deploy/service.yaml", "replicas: 1\n")
	f.write(".kogen/project.yaml", "version: 1\ncommands:\n  checks: [{name: unit, run: 'true'}]\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\nprotect: ['config/*.cfg']\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "Configure a protected glob")
}

func kbrJourney(t *testing.T, mode string) kbrObs {
	var f *shpFx
	request := "Change the deployment check."
	switch mode {
	case "unprotected":
		f = newShpFx(t, kbrBlocked("b1", []any{"deploy/service.yaml", "calc.js", "README.md"}, "deployment files look guarded"), shpSubmit("c1", shpFiles()))
		kbrProtect(f)
		f.setReport("PASSED", 0)
		request = "Add calc.is_even(n)."
	case "mixed":
		f = newShpFx(t,
			kbrBlocked("b1", []any{"calc.js", "config/deploy.cfg", "README.md"}, "the deployment check must change"),
			kbrBlocked("b2", []any{"config/deploy.cfg"}, "the deployment check definition must change"))
		kbrProtect(f)
	case "protected":
		f = newShpFx(t, kbrBlocked("b1", []any{"config/deploy.cfg", ".kogen/project.yaml"}, "the deployment check and its configuration must change"))
		kbrProtect(f)
	case "malformed":
		f = newShpFx(t, kbrBlocked("b1", []any{}, 7), kbrBlocked("b2", []any{" "}, "x"),
			shpSubmit("q1", map[string]any{"questions": []any{map[string]any{"question": "Continue?"}}}), shpUI(1))
		kbrProtect(f)
	case "automatic":
		f = newShpFx(t, kbrBlocked("b1", []any{"calc.js"}, "the calculator must change"), shpSubmit("c1", shpFiles()))
		f.setReport("PASSED", 0)
		request = "Add calc.is_even(n)."
	default:
		t.Fatalf("unknown KBR_MODE %q", mode)
	}
	head := f.head()
	exit, err := f.shape(request)
	o := kbrObs{outputs: map[string]string{}, exit: exit, stdout: f.Stdout.String()}
	if err != nil {
		o.err = err.Error()
	}
	o.requests = len(shpShaperRequests(f.Prov.requests()))
	for _, req := range f.Prov.requests() {
		for id, out := range shpOutputs(req) {
			o.outputs[id] = out
		}
	}
	o.published = f.exists(".kogen/intents/is-even/intent.md")
	o.written = f.exists(".kogen/intents/is-even") || f.exists(".kogen/local/approvals/is-even.json") || f.exists(".kogen/local/runs")
	o.headMoved = f.head() != head
	o.status = f.git("status", "--porcelain")
	o.cfg = "strict=true\n"
	if f.exists("config/deploy.cfg") {
		o.cfg = f.read("config/deploy.cfg")
	}
	return o
}

func TestKShapeBlockedReportRepairAcceptance(t *testing.T) {
	mode := os.Getenv("KBR_MODE")
	switch mode {
	case "good":
		for m, o := range kbrGood() {
			for _, p := range kbrJudge(m, o) {
				t.Errorf("known-good %s observation rejected: %s", m, p)
			}
		}
	case "bad":
		var rejected []string
		for _, b := range kbrBad() {
			p := kbrJudge(b.mode, b.o)
			if len(p) == 0 {
				fmt.Printf("known-bad %s observation accepted: %s\n", b.mode, b.name)
				fmt.Println("KBR-OK")
				return
			}
			rejected = append(rejected, b.mode+": "+b.name+": "+p[0])
		}
		t.Errorf("all %d known-bad observations rejected:\n%s", len(rejected), strings.Join(rejected, "\n"))
	default:
		for _, p := range kbrJudge(mode, kbrJourney(t, mode)) {
			t.Error(p)
		}
	}
	if !t.Failed() {
		fmt.Println("KBR-OK")
	}
}
`;
let built = null;
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-shape-blocked-report-repair-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const root = process.cwd();
  const src = path.join(dir, 'journey_test.go');
  fs.writeFileSync(src, GO);
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.join(root, 'shape/k_shape_blocked_report_repair_acceptance_test.go')]: src}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(''), encoding: 'utf8', timeout: 400000});
  built = {bin, error: r.error || r.status !== 0 ? `${TEST}: go test -c -vet=off -overlay ./shape failed (exit ${r.status}): ${r.error || ''}\n${r.stdout}${r.stderr}` : ''};
  return built;
}
function env(mode) {
  const e = {...process.env, KBR_MODE: mode};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}
function run(mode) {
  const b = build();
  if (b.error) return {ok: false, detail: b.error};
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=120s'];
  const r = spawnSync(b.bin, args, {cwd: path.join(process.cwd(), 'shape'), env: env(mode), encoding: 'utf8', timeout: 150000, maxBuffer: 8 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  const ran = text.includes('--- PASS: ' + TEST + ' ');
  const ok = !r.error && r.status === 0 && ran && text.includes('KBR-OK\n') && !text.includes('--- SKIP:');
  return {ok, detail: `${TEST} (KBR_MODE=${mode}): ${b.bin} ${args.join(' ')} built by go test -c -vet=off -overlay ./shape; exit=${r.status}; error=${r.error || ''}; absent, skipped or zero runs=${!ran}\n${text}`};
}
if (require.main === module) {
  const r = run(process.argv[2]);
  console.log(r.ok ? 'accepted' : 'rejected');
  if (process.argv[3] === '--diagnostic') console.error(r.detail);
  process.exitCode = r.ok ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('k-shape-blocked-report-repair: the blocked report journey runs in {string} mode', {timeout: 600000}, function (mode) { this.kbrResult = run(mode); });
  Then('k-shape-blocked-report-repair: the blocked report journey satisfies the contract', function () { assert.ok(this.kbrResult.ok, this.kbrResult.detail); });
}
