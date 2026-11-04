package consumes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
)

func TestBaselineAndHash(t *testing.T) {
	dir, cfg := newProject(t)
	baseline, err := project.Baseline(dir, slug, cfg)
	if err != nil {
		t.Fatalf("project.Baseline: %v", err)
	}
	// Card order: intent.md, acceptance.feature, the rest of the Acceptance list, the other
	// files of the Intent's folder, the configuration, then the protected files, each once.
	want := []string{
		".kogen/intents/widget/intent.md",
		".kogen/intents/widget/acceptance.feature",
		"features/steps/widget__steps.js",
		".kogen/intents/widget/notes.md",
		".kogen/project.yaml",
		".golangci.yml",
		"testdata/a/golden.txt",
	}
	if !slices.Equal(baseline, want) {
		t.Errorf("project.Baseline = %q, want %q", baseline, want)
	}

	hash := project.Hash(dir, baseline)
	sorted := slices.Sorted(slices.Values(baseline))
	var records strings.Builder
	for _, p := range sorted {
		b, err := os.ReadFile(filepath.Join(dir, p))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(b)
		records.WriteString(p + "\x00" + hex.EncodeToString(sum[:]) + "\n")
	}
	total := sha256.Sum256([]byte(records.String()))
	if want := hex.EncodeToString(total[:]); hash != want {
		t.Errorf("project.Hash = %s, want the SHA-256 of the sorted path\\0digest records: %s", hash, want)
	}
	if got := project.Hash(dir, slices.Sorted(slices.Values(baseline))); got != hash {
		t.Errorf("project.Hash depends on the order of the paths: %s and %s", got, hash)
	}

	// Changing a baselined file changes the hash; a file outside the baseline does not.
	for _, p := range []string{".kogen/intents/widget/acceptance.feature", "features/steps/widget__steps.js", ".kogen/project.yaml", ".golangci.yml"} {
		orig, _ := os.ReadFile(filepath.Join(dir, p))
		write(t, dir, p, string(orig)+"# edited\n")
		if got := project.Hash(dir, baseline); got == hash {
			t.Errorf("project.Hash did not change when the baselined file %s changed", p)
		}
		write(t, dir, p, string(orig))
		if got := project.Hash(dir, baseline); got != hash {
			t.Errorf("project.Hash after restoring %s = %s, want %s again", p, got, hash)
		}
	}
	write(t, dir, "README.md", "outside the baseline, edited\n")
	if got := project.Hash(dir, baseline); got != hash {
		t.Errorf("project.Hash changed with a file outside the baseline")
	}
	// A file that cannot be read hashes as "missing", and differs from an empty file.
	if err := os.Remove(filepath.Join(dir, ".kogen/intents/widget/notes.md")); err != nil {
		t.Fatal(err)
	}
	missing := project.Hash(dir, baseline)
	write(t, dir, ".kogen/intents/widget/notes.md", "")
	if missing == hash || missing == project.Hash(dir, baseline) {
		t.Errorf("project.Hash must tell a missing file from a changed and from an empty one")
	}

	// A new file in the Intent's folder joins the baseline, and a file Git ignores does not.
	write(t, dir, ".kogen/intents/widget/model.qnt", "module m {}\n")
	grown, err := project.Baseline(dir, slug, cfg)
	if err != nil || !slices.Contains(grown, ".kogen/intents/widget/model.qnt") {
		t.Errorf("project.Baseline = %q, %v, want the new file of the Intent folder", grown, err)
	}
	write(t, dir, "testdata/ignored.log", "x\n")
	if again, _ := project.Baseline(dir, slug, cfg); slices.Contains(again, "testdata/ignored.log") {
		t.Errorf("project.Baseline lists the ignored file testdata/ignored.log")
	}
}

func TestProtected(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, ".gitignore", "*.log\n")
	write(t, dir, ".golangci.yml", "a\n")
	write(t, dir, "docs/a.md", "a\n")
	write(t, dir, "docs/sub/b.md", "b\n")
	write(t, dir, "go.lock", "l\n")
	write(t, dir, "sub/x.lock", "l\n")
	write(t, dir, "testdata/g.txt", "g\n")
	write(t, dir, "other.txt", "o\n")
	commitAll(t, dir, "base")
	write(t, dir, "testdata/untracked.txt", "u\n") // untracked, not ignored: protected
	write(t, dir, "testdata/ignored.log", "i\n")   // ignored: never protected

	cfg := &project.Config{Protect: []string{"testdata/**", ".golangci.yml", "docs/", "*.lock", "testdata/g.txt"}}
	got, err := project.Protected(dir, cfg)
	if err != nil {
		t.Fatalf("project.Protected: %v", err)
	}
	want := []string{".golangci.yml", "docs/a.md", "docs/sub/b.md", "go.lock", "testdata/g.txt", "testdata/untracked.txt"}
	if !slices.Equal(got, want) {
		t.Errorf("project.Protected = %q, want the tracked and untracked-not-ignored matches, sorted, without duplicates: %q", got, want)
	}
	if got, err := project.Protected(dir, &project.Config{}); err != nil || len(got) != 0 {
		t.Errorf("project.Protected without protect = %q, %v, want none", got, err)
	}
	if _, err := project.Protected(t.TempDir(), cfg); err == nil {
		t.Errorf("project.Protected outside a repository must fail (git ls-files)")
	}
}

func TestApprovalBytes(t *testing.T) {
	full := project.Approval{
		Slug: "widget", SHA256: "abc", Base: "deadbeef", Approver: "Ada <ada@example.invalid>",
		ApprovedAt:       "2026-11-04T10:00:00Z",
		SourceProvenance: map[string]any{"sha256": "s", "path": "/p"},
	}
	got, err := project.EncodeApproval(full)
	want := `{"slug":"widget","sha256":"abc","base":"deadbeef","approver":"Ada <ada@example.invalid>","approved_at":"2026-11-04T10:00:00Z","source_provenance":{"path":"/p","sha256":"s"}}` + "\n"
	if err != nil || string(got) != want {
		t.Errorf("project.EncodeApproval = %q, %v, want %q", got, err, want)
	}
	// No base: the key is left out; no receipt (legacy): null.
	got, err = project.EncodeApproval(project.Approval{Slug: "w", SHA256: "x", Approver: "Me", ApprovedAt: "t"})
	want = `{"slug":"w","sha256":"x","approver":"Me","approved_at":"t","source_provenance":null}` + "\n"
	if err != nil || string(got) != want {
		t.Errorf("project.EncodeApproval without base = %q, %v, want %q", got, err, want)
	}
}

func TestReadApproval(t *testing.T) {
	dir := t.TempDir()
	if got, err := project.ReadApproval(dir, slug); got != nil || err != nil {
		t.Errorf("project.ReadApproval without a file = %v, %v, want nil, nil", got, err)
	}
	stored := project.Approval{
		Slug: slug, SHA256: "0123", Base: "cafe", Approver: "Grace Hopper <grace@example.invalid>",
		ApprovedAt: "2026-11-04T11:00:00Z", SourceProvenance: map[string]any{"path": "/p", "sha256": "s"},
	}
	b, err := project.EncodeApproval(stored)
	if err != nil {
		t.Fatal(err)
	}
	write(t, dir, ".kogen/local/approvals/widget.json", string(b))
	got, err := project.ReadApproval(dir, slug)
	if err != nil || got == nil {
		t.Fatalf("project.ReadApproval: %v, %v", got, err)
	}
	eq(t, "project.Approval.Slug", got.Slug, stored.Slug)
	eq(t, "project.Approval.SHA256", got.SHA256, stored.SHA256)
	eq(t, "project.Approval.Base", got.Base, stored.Base)
	eq(t, "project.Approval.Approver", got.Approver, stored.Approver)
	eq(t, "project.Approval.ApprovedAt", got.ApprovedAt, stored.ApprovedAt)
	eq(t, "project.Approval.SourceProvenance", got.SourceProvenance, map[string]any{"path": "/p", "sha256": "s"})
	// A marker that is not an approval is an error, never absence.
	write(t, dir, ".kogen/local/approvals/broken.json", "not json")
	if got, err := project.ReadApproval(dir, "broken"); got != nil || err == nil {
		t.Errorf("project.ReadApproval of invalid JSON = %v, %v, want an error", got, err)
	}
	// A file that cannot be read is an error too.
	if err := os.MkdirAll(filepath.Join(dir, ".kogen/local/approvals/dir.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := project.ReadApproval(dir, "dir"); got != nil || err == nil {
		t.Errorf("project.ReadApproval of a folder = %v, %v, want an error", got, err)
	}
}

func TestEncodeJSON(t *testing.T) {
	// The bytes of the records Shape writes: keys in byte order, compact, no HTML escaping.
	got, err := project.EncodeJSON(map[string]any{
		"z": []string{"a", "b"}, "a": nil, "m": true, "n": 42, "big": int64(1 << 40),
		"nested": map[string]any{"y": []any{"x", 1, false}, "b": map[string]string{"k": "v", "a": "w"}},
		"text":   "quote \" backslash \\ newline \n tab \t ctl \x01 <html> & é ✓",
		"rows":   []map[string]any{{"b": 1, "a": 2}},
		"num":    json.Number("1.50"),
	})
	want := `{"a":null,"big":1099511627776,"m":true,"n":42,"nested":{"b":{"a":"w","k":"v"},"y":["x",1,false]},"num":1.50,"rows":[{"a":2,"b":1}],"text":"quote \" backslash \\ newline \n tab \t ctl \u0001 <html> & é ✓","z":["a","b"]}`
	if err != nil || string(got) != want {
		t.Errorf("project.EncodeJSON =\n%s\n%v\nwant\n%s", got, err, want)
	}

	// A struct is encoded through its tags and then in key order.
	got, err = project.EncodeJSON(project.Approval{Slug: "s", SHA256: "h", Approver: "a", ApprovedAt: "t"})
	want = `{"approved_at":"t","approver":"a","sha256":"h","slug":"s","source_provenance":null}`
	if err != nil || string(got) != want {
		t.Errorf("project.EncodeJSON of a struct = %s, %v, want %s", got, err, want)
	}
	got, err = project.EncodeJSON(project.Role{Provider: "anthropic", Model: "m", Effort: "high"})
	want = `{"Effort":"high","Model":"m","Provider":"anthropic"}`
	if err != nil || string(got) != want {
		t.Errorf("project.EncodeJSON of a project.Role = %s, %v, want %s (empty BaseURL, Executor and MaxRounds omitted)", got, err, want)
	}

	// A string that is not UTF-8 is an error, not replacement characters.
	if got, err := project.EncodeJSON(map[string]any{"bad": "\xff"}); err == nil {
		t.Errorf("project.EncodeJSON of invalid UTF-8 = %s, want an error", got)
	}
	// The scalars alone.
	for _, c := range []struct {
		in   any
		want string
	}{{nil, "null"}, {"s", `"s"`}, {7, "7"}, {false, "false"}, {[]string(nil), "[]"}} {
		if got, err := project.EncodeJSON(c.in); err != nil || string(got) != c.want {
			t.Errorf("project.EncodeJSON(%#v) = %s, %v, want %s", c.in, got, err, c.want)
		}
	}
}
