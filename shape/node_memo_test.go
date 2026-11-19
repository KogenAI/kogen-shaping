package shape

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

// Loading Cucumber in a new Node process costs a quarter to most of a CPU second, and the shape
// tests start it hundreds of times for the same few step files. What such a run reports is a
// function of the files it loads, so the first run of each kind in this test process is
// recorded and a later run over the same loadable content replays it. A run whose report names
// the directory it ran in is not recorded, and neither is one that failed or timed out; the
// tests of the control run and of the step self-check that assert on a real run's failure still
// get it.

// memo is a set of recorded runs by key.
type memo struct{ m sync.Map }

func (c *memo) load(key string) ([]byte, bool) {
	v, ok := c.m.Load(key)
	if !ok {
		return nil, false
	}
	return v.([]byte), true
}

func (c *memo) store(key string, b []byte) { c.m.Store(key, b) }

var stepScriptMemo, bindingMemo, controlMemo memo

// loadableFingerprint identifies what a Node run in root can load: the path of every file, and
// the bytes of the scripts, data and features among them. Git's and the shaping state's own
// files, and installed packages, are left out.
func loadableFingerprint(root string) string {
	h := sha256.New()
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			if rel == ".git" || rel == filepath.Join(".kogen", "local") || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		h.Write([]byte(rel + "\x00"))
		switch filepath.Ext(path) {
		case ".js", ".cjs", ".mjs", ".json", ".feature":
			if b, err := os.ReadFile(path); err == nil {
				h.Write(b)
			}
		}
		h.Write([]byte{0})
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))
}

// standardNode is whether node is the Node the suite resolved: a test that runs its own wrapper
// script around Node counts or changes the launches, so its runs are never replayed.
func standardNode(node string) bool { return node != "" && node == os.Getenv("KOGEN_NODE") }

func memoKey(parts ...any) string {
	b, _ := json.Marshal(parts)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func init() {
	stepScript := runStepScript
	runStepScript = func(ctx context.Context, node, cucumber, root string, env, files, texts []string) (verify.StepRaw, bool, error) {
		if !standardNode(node) {
			return stepScript(ctx, node, cucumber, root, env, files, texts)
		}
		key := memoKey("step", node, cucumber, loadableFingerprint(root), files, texts)
		if b, ok := stepScriptMemo.load(key); ok {
			var e struct {
				Raw    verify.StepRaw
				Loaded bool
			}
			if json.Unmarshal(b, &e) == nil {
				return e.Raw, e.Loaded, nil
			}
		}
		raw, loaded, err := stepScript(ctx, node, cucumber, root, env, files, texts)
		if err == nil {
			if b, merr := json.Marshal(struct {
				Raw    verify.StepRaw
				Loaded bool
			}{raw, loaded}); merr == nil && !bytes.Contains(b, []byte(root)) {
				stepScriptMemo.store(key, b)
			}
		}
		return raw, loaded, err
	}

	bindings := runBindingJobs
	runBindingJobs = func(ctx context.Context, node, cucumber, root string, env []string, jobs []bindingJob) ([]bindingResult, error) {
		if !standardNode(node) {
			return bindings(ctx, node, cucumber, root, env, jobs)
		}
		type wire struct{ Files, Texts []string }
		type recorded struct {
			Matches [][]string
			Loaded  bool
		}
		var in []wire
		for _, j := range jobs {
			in = append(in, wire{j.files, j.texts})
		}
		key := memoKey("binding", node, cucumber, loadableFingerprint(root), in)
		if b, ok := bindingMemo.load(key); ok {
			var rec []recorded
			if json.Unmarshal(b, &rec) == nil && len(rec) == len(jobs) {
				out := make([]bindingResult, len(rec))
				for i, r := range rec {
					out[i] = bindingResult{r.Matches, r.Loaded}
				}
				return out, nil
			}
		}
		res, err := bindings(ctx, node, cucumber, root, env, jobs)
		if err == nil {
			rec := make([]recorded, len(res))
			for i, r := range res {
				rec[i] = recorded{r.matches, r.loaded}
			}
			if b, merr := json.Marshal(rec); merr == nil && !bytes.Contains(b, []byte(root)) {
				bindingMemo.store(key, b)
			}
		}
		return res, err
	}

	control := unpassableRunCommand
	unpassableRunCommand = func(ctx context.Context, s *sys.Session, command, dir string, opts sys.Options) (sys.Result, error) {
		// The run is in <base>/tree and writes to <base>/kit; both paths are new for every run.
		base := filepath.Dir(dir)
		const here = "\x00base\x00"
		norm := func(text string) string { return strings.ReplaceAll(text, base, here) }
		env := map[string]string{}
		for k, v := range opts.Env {
			env[k] = norm(v)
		}
		log := opts.Env["KOGEN_UNPASSABLE_LOG"]
		report := filepath.Join(filepath.Dir(log), "report.ndjson")
		if log == "" || filepath.Dir(log) != filepath.Join(base, "kit") || !strings.HasPrefix(command, ShellQuote(os.Getenv("KOGEN_NODE"))+" ") {
			return control(ctx, s, command, dir, opts)
		}
		key := memoKey("control", norm(command), env, opts.Timeout, loadableFingerprint(dir))
		type recorded struct {
			Status       int
			Output       string
			Report, Uses *string
		}
		if b, ok := controlMemo.load(key); ok {
			var rec recorded
			if json.Unmarshal(b, &rec) == nil {
				for path, text := range map[string]*string{report: rec.Report, log: rec.Uses} {
					if text != nil {
						if err := os.WriteFile(path, []byte(strings.ReplaceAll(*text, here, base)), 0o644); err != nil {
							return control(ctx, s, command, dir, opts)
						}
					}
				}
				return sys.Result{Status: rec.Status, Output: strings.ReplaceAll(rec.Output, here, base)}, nil
			}
		}
		res, err := control(ctx, s, command, dir, opts)
		if err != nil || res.TimedOut || res.Rule != "" {
			return res, err
		}
		rec := recorded{Status: res.Status, Output: norm(res.Output)}
		for path, into := range map[string]**string{report: &rec.Report, log: &rec.Uses} {
			if b, err := os.ReadFile(path); err == nil {
				text := norm(string(b))
				*into = &text
			}
		}
		if b, err := json.Marshal(rec); err == nil {
			controlMemo.store(key, b)
		}
		return res, nil
	}
}
