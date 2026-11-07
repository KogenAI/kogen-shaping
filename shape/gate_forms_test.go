package shape

import (
	"fmt"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
)

// TestEvidenceTrackedPathsThenProse: an item that is not a whole tracked path cites its leading
// tracked words; the refused item kinds stay refused.
func TestEvidenceTrackedPathsThenProse(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.write("src/calc.py", "x = 1\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "calc")
	for _, c := range []struct{ item, want string }{
		{"README.md and the check runs it", ""},
		{"README.md, `src/calc.py` and the check runs it", ""},
		{"features/steps/ hold the steps", ""},
		{"README.md and src/new.py", "- decision 1: src/new.py is not a file or directory tracked at HEAD"},
		{"README.md and (\"src/new.py\"), too", "- decision 1: src/new.py is not a file or directory tracked at HEAD"},
		{"README.md and src/calc.py.", ""},
		{"README.md and the src/ folder", ""},
		{"see README.md", "- decision 1: see README.md is not a file or directory tracked at HEAD"},
		{"src/missing.py and README.md", "- decision 1: src/missing.py and README.md is not a file or directory tracked at HEAD"},
		{"calc.add", "- decision 1: calc.add is not a file or directory tracked at HEAD"},
		{"src/missing.py", "- decision 1: src/missing.py is not a file or directory tracked at HEAD"},
	} {
		sh := &shaping{repo: f.Repo, ext: &shapeExec{t: t}, inspected: true}
		got, err := sh.evidenceProblems(fmt.Sprintf("## Decisions\n- Zero is even. (evidence: %s) (source: request)\n", c.item))
		shpEq(t, "err", err, nil)
		shpEq(t, c.item, strings.Join(got, "\n"), c.want)
	}
	// The cited paths of such an item still need to be read by a native shaping.
	sh := &shaping{repo: f.Repo, readPaths: map[string]bool{"README.md": true}, probed: map[string]bool{}}
	got, err := sh.evidenceProblems("## Decisions\n- Zero is even. (evidence: README.md src/calc.py and prose) (source: request)\n")
	shpEq(t, "err", err, nil)
	shpEq(t, "unread", strings.Join(got, "\n"), "- decision 1: this shaping did not read src/calc.py")
}

// TestValidateFilesIntentFolderBounds: shaping may write up to 16 files of at most 65536 bytes
// under the Intent's own folder besides its four own files, counted over the accumulated proposals.
func TestValidateFilesIntentFolderBounds(t *testing.T) {
	parallel(t)
	const dir = ".kogen/intents/is-even/"
	file := func(p string, size int) any {
		return map[string]any{"path": p, "content": strings.Repeat("x", size)}
	}
	files := func(prefix string, n int) []any {
		var out []any
		for i := 0; i < n; i++ {
			out = append(out, file(fmt.Sprintf("%sfixtures/%s%02d.txt", dir, prefix, i), 1))
		}
		return out
	}
	const countLine = "shaping may write at most 16 files under .kogen/intents/is-even/ besides intent.md, acceptance.feature, model.qnt and controls.json"
	proposed := map[string]string{}
	for _, f := range files("p", 10) {
		proposed[f.(map[string]any)["path"].(string)] = "x"
	}
	proposed[dir+"intent.md"] = "x"
	for _, c := range []struct {
		name  string
		held  map[string]string
		files []any
		want  string
	}{
		{"16 files", nil, files("a", 16), ""},
		{"a file of 65536 bytes", nil, []any{file(dir+"fixtures/big.txt", 65536)}, ""},
		{"17 files", nil, files("a", 17), countLine},
		{"6 files beside 10 proposed", proposed, files("a", 6), ""},
		{"7 files beside 10 proposed", proposed, files("a", 7), countLine},
		{"a proposed path again", proposed, append(files("p", 10), files("a", 6)...), ""},
		{"a file of 65537 bytes", nil, []any{file(dir+"fixtures/big.txt", 65537)}, dir + "fixtures/big.txt: a file under .kogen/intents/is-even/ may hold at most 65536 bytes"},
		{"another Intent's folder", nil, []any{file(".kogen/intents/other/x.txt", 1)}, ".kogen/intents/other/x.txt: shaping may write only files under .kogen/intents/is-even/ and step files under commands.acceptance.steps_dir (unset)/"},
		{"a path that is not normal", nil, []any{file(dir+"../other/x.txt", 1)}, dir + "../other/x.txt: paths must be relative and normal"},
	} {
		held := map[string]string{}
		for k, v := range c.held {
			held[k] = v
		}
		sh := &shaping{slug: "is-even", proposed: held, cfg: &project.Config{}}
		_, problems, err := sh.validateFiles(c.files)
		shpEq(t, "err", err, nil)
		shpEq(t, c.name, strings.Join(problems, "\n"), c.want)
	}
}
