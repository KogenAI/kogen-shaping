package consumes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/git"
	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
)

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// receipt decodes the current receipt of the widget Intent.
func receipt(t *testing.T, dir string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(project.SourcePath(dir, slug))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

// shaped is a project whose widget Intent has a captured receipt, and the reference BindSource gives.
func shaped(t *testing.T) (string, *project.Config, *project.SourceRef) {
	t.Helper()
	dir, cfg := newProject(t)
	origin, err := project.CaptureOrigin(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	src := project.ShapeSource{Request: "Make widgets.\n", SourcePath: "request.txt", Origin: origin, Rounds: []map[string]any{{
		"questions": []any{map[string]any{"text": "Which colour?", "options": []any{map[string]any{"label": "Red"}, map[string]any{"label": "Blue"}}}},
		"answers":   map[string]any{"1": "b", "2": "free text"},
	}}}
	if err := project.CaptureSource(dir, slug, cfg, src); err != nil {
		t.Fatalf("project.CaptureSource: %v", err)
	}
	ref, err := project.BindSource(dir, slug)
	if err != nil || ref == nil {
		t.Fatalf("project.BindSource: %v, %v", ref, err)
	}
	return dir, cfg, ref
}

func wantCustodyStop(t *testing.T, what string, err error) {
	t.Helper()
	se, ok := sys.AsError(err)
	if !ok || se.Class != "candidate" || se.Reason != "source_provenance_changed" || se.Exit != 1 {
		t.Errorf("%s = %v, want candidate/source_provenance_changed exit 1", what, err)
	}
}

func TestSourceComparison(t *testing.T) {
	dir, _ := newProject(t)
	got, err := project.SourceComparison(dir, slug)
	if err != nil || got != "The request maps to the widget." {
		t.Errorf("project.SourceComparison = %q, %v, want the trimmed section body", got, err)
	}
	// The section ends at the next ## heading and is trimmed.
	write(t, dir, project.IntentMDPath(slug), "# T\n\n## Source comparison  \n\n  first line\n  second line  \n\n## Tests\n- x\n")
	if got, err := project.SourceComparison(dir, slug); err != nil || got != "first line\n  second line" {
		t.Errorf("project.SourceComparison = %q, %v, want the body up to the next heading, trimmed", got, err)
	}
	for name, text := range map[string]string{
		"missing":  "# T\n\n## Tests\n- x\n",
		"empty":    "# T\n\n## Source comparison\n\n\n## Tests\n",
		"too long": "# T\n\n## Source comparison\n" + strings.Repeat("x", 8193) + "\n",
	} {
		write(t, dir, project.IntentMDPath(slug), text)
		got, err := project.SourceComparison(dir, slug)
		se, ok := sys.AsError(err)
		if !ok || got != "" || se.Class != "intent" || se.Reason != "lint" || se.Exit != 1 || !strings.Contains(se.Detail, "## Source comparison") {
			t.Errorf("project.SourceComparison of a %s section = %q, %v, want intent/lint exit 1", name, got, err)
		}
	}
	write(t, dir, project.IntentMDPath(slug), "# T\n\n## Source comparison\n"+strings.Repeat("x", 8192)+"\n")
	if got, err := project.SourceComparison(dir, slug); err != nil || len(got) != 8192 {
		t.Errorf("project.SourceComparison of exactly 8192 bytes = %d bytes, %v, want it whole", len(got), err)
	}
	if _, err := project.SourceComparison(t.TempDir(), slug); err == nil {
		t.Errorf("project.SourceComparison without intent.md must fail")
	}
}

func TestCaptureOrigin(t *testing.T) {
	dir, cfg := newProject(t)
	origin, err := project.CaptureOrigin(dir, cfg)
	if err != nil {
		t.Fatalf("project.CaptureOrigin: %v", err)
	}
	head := git.New(dir, nil).Head()
	eq(t, "project.SourceOrigin.Head", origin.Head, head)
	eq(t, "project.SourceOrigin.Tree", origin.Tree, treeOf(t, dir, "HEAD"))
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(origin.Configuration) {
		t.Errorf("project.SourceOrigin.Configuration = %q, want a SHA-256 hex digest", origin.Configuration)
	}
	// The tree is the checkout as it is now: an untracked file joins it, an ignored one does not.
	write(t, dir, "scratch/new.txt", "n\n")
	write(t, dir, "scratch/skip.log", "n\n")
	now, err := project.CaptureOrigin(dir, cfg)
	if err != nil || now.Head != head || now.Tree == origin.Tree || now.Configuration != origin.Configuration {
		t.Errorf("project.CaptureOrigin with an untracked file = %+v, %v, want the same Head and configuration and a new Tree", now, err)
	}
	if listed := runGit(t, dir, "ls-tree", "-r", "--name-only", now.Tree); !strings.Contains(listed, "scratch/new.txt") || strings.Contains(listed, "skip.log") {
		t.Errorf("project.CaptureOrigin tree lists %q, want the untracked file and not the ignored one", listed)
	}
	// The configuration digest follows the configuration.
	changed := *cfg
	changed.Setup = []string{"something else"}
	if other, err := project.CaptureOrigin(dir, &changed); err != nil || other.Configuration == origin.Configuration {
		t.Errorf("project.CaptureOrigin digest = %q, %v, want it to change with the setup commands", other.Configuration, err)
	}
	// Without a commit there is no origin.
	empty := newRepo(t)
	if _, err := project.CaptureOrigin(empty, cfg); err == nil {
		t.Errorf("project.CaptureOrigin of a repository without commits must fail")
	}
}

func TestCaptureSource(t *testing.T) {
	dir, cfg, ref := shaped(t)
	doc := receipt(t, dir)
	baseline, err := project.Baseline(dir, slug, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if doc["version"] != float64(1) {
		t.Errorf("project.CaptureSource: version = %v, want 1", doc["version"])
	}
	req, _ := doc["request"].(map[string]any)
	eq(t, "project.ShapeSource.Request as the receipt's request.text", req["text"], "Make widgets.\n")
	eq(t, "project.ShapeSource.SourcePath as the receipt's request.source_path", req["source_path"], "request.txt")
	eq(t, "the SHA-256 of project.ShapeSource.Request as the receipt's request.sha256", req["sha256"], sha([]byte("Make widgets.\n")))
	eq(t, "project.CaptureSource contract_sha256", doc["contract_sha256"], project.Hash(dir, baseline))
	eq(t, "project.SourceRef.ContractSHA256 of project.BindSource", ref.ContractSHA256, project.Hash(dir, baseline))
	if doc["comparison"] != "The request maps to the widget." {
		t.Errorf("project.CaptureSource: comparison = %v, want the Intent's Source comparison", doc["comparison"])
	}
	if c, ok := doc["corrections"].([]any); !ok || len(c) != 0 {
		t.Errorf("project.CaptureSource: corrections = %v, want an empty list", doc["corrections"])
	}
	origin, _ := doc["origin"].(map[string]any)
	eq(t, "project.ShapeSource.Origin as the receipt's origin.head (project.SourceOrigin.Head)", origin["head"], git.New(dir, nil).Head())
	eq(t, "project.ShapeSource.Origin as the receipt's origin.tree (project.SourceOrigin.Tree)", origin["tree"], treeOf(t, dir, "HEAD"))
	if c, _ := origin["configuration"].(string); len(c) != 64 {
		t.Errorf("project.ShapeSource.Origin as the receipt's origin.configuration (project.SourceOrigin.Configuration) = %v, want a SHA-256 digest", origin["configuration"])
	}
	rounds, _ := doc["rounds"].([]any)
	if len(rounds) != 1 {
		t.Fatalf("project.ShapeSource.Rounds as the receipt's rounds = %v, want the one round given", doc["rounds"])
	}
	round := rounds[0].(map[string]any)
	resolved, _ := round["resolved_answers"].(map[string]any)
	eq(t, "project.ShapeSource.Rounds: a single-letter answer resolved to its option's label", resolved["1"], "Blue")
	eq(t, "project.ShapeSource.Rounds: any other answer kept as typed", resolved["2"], "free text")
	if answers, _ := round["answers"].(map[string]any); answers["1"] != "b" {
		t.Errorf("project.CaptureSource: answers = %v, want the answers kept as given", answers)
	}

	// Custody: private folders and file, a ref that names the path and the bytes.
	path := project.SourcePath(dir, slug)
	for p, want := range map[string]os.FileMode{
		path: 0o600, filepath.Dir(path): 0o700, filepath.Dir(filepath.Dir(path)): 0o700,
	} {
		if st, err := os.Lstat(p); err != nil || st.Mode().Perm() != want {
			t.Errorf("project.CaptureSource: %s has mode %v, %v, want %v", p, st.Mode().Perm(), err, want)
		}
	}
	b, _ := os.ReadFile(path)
	eq(t, "project.SourceRef.Path of project.BindSource", ref.Path, path)
	eq(t, "the SHA-256 recorded by project.BindSource", ref.SHA256, sha(b))
	if !strings.HasSuffix(string(b), "\n") {
		t.Errorf("project.CaptureSource: the receipt does not end with a newline: %q", b)
	}

	// Rounds with an answer that is not text, or a request that is not UTF-8, are refused.
	bad := project.ShapeSource{Request: "x", Rounds: []map[string]any{{"answers": map[string]any{"1": 5}}}}
	if err := project.CaptureSource(dir, slug, cfg, bad); err == nil {
		t.Errorf("project.CaptureSource with a non-text answer must fail")
	}
	if err := project.CaptureSource(dir, slug, cfg, project.ShapeSource{Request: "\xff"}); err == nil {
		t.Errorf("project.CaptureSource with a request that is not UTF-8 must fail")
	}
	// Standard input is recorded as "-".
	if err := project.CaptureSource(dir, slug, cfg, project.ShapeSource{Request: "from stdin"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := receipt(t, dir)["request"].(map[string]any); got["source_path"] != "-" {
		t.Errorf("project.CaptureSource of a request without a path = %v, want source_path \"-\"", got)
	}
	// The earlier receipt stays, byte for byte, under its hash.
	if kept, err := os.ReadFile(ref.Archived().Path); err != nil || sha(kept) != ref.SHA256 {
		t.Errorf("project.CaptureSource must keep the replaced receipt at %s: %v", ref.Archived().Path, err)
	}
}

func TestNewSourceRequest(t *testing.T) {
	r := project.NewSourceRequest("fix it\n", "fix.txt")
	if r.Text != "fix it\n" || r.SourcePath != "fix.txt" || r.SHA256 != sha([]byte("fix it\n")) {
		t.Errorf("project.NewSourceRequest = %+v, want the text, its path and the SHA-256 of the text", r)
	}
	var s project.SourceRequest = project.NewSourceRequest("from stdin", "")
	if s.SourcePath != "-" {
		t.Errorf("project.NewSourceRequest without a path: SourcePath = %q, want \"-\"", s.SourcePath)
	}
}

func TestBindSource(t *testing.T) {
	dir, _ := newProject(t)
	ref, err := project.BindSource(dir, slug)
	if ref != nil || err != nil {
		t.Errorf("project.BindSource of a legacy Intent = %v, %v, want nil, nil", ref, err)
	}
	// A provenance folder without a receipt is refused, never read as absence.
	if err := os.MkdirAll(filepath.Dir(project.SourcePath(dir, slug)), 0o700); err != nil {
		t.Fatal(err)
	}
	ref, err = project.BindSource(dir, slug)
	wantCustodyStop(t, "project.BindSource of a folder without a receipt", err)
	if ref != nil {
		t.Errorf("project.BindSource returned %v with an error", ref)
	}

	shapedDir, _, bound := shaped(t)
	if again, err := project.BindSource(shapedDir, slug); err != nil || !again.Equal(bound) {
		t.Errorf("project.BindSource twice = %v, %v, want the same binding as %v", again, err, bound)
	}
	for name, content := range map[string]string{"not json": "oops\n", "not an object": "[]\n", "version 2": `{"version":2,"contract_sha256":"x"}` + "\n", "no contract": `{"version":1}` + "\n"} {
		d, _, _ := shaped(t)
		p := project.SourcePath(d, slug)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if got, err := project.BindSource(d, slug); got != nil || err == nil {
			t.Errorf("project.BindSource of a receipt that is %s = %v, %v, want a refusal", name, got, err)
		} else {
			wantCustodyStop(t, "project.BindSource of a receipt that is "+name, err)
		}
	}
}

func TestValidateSource(t *testing.T) {
	if err := project.ValidateSource(nil); err != nil {
		t.Errorf("project.ValidateSource(nil) = %v, want nil for a legacy Intent", err)
	}
	dir, _, ref := shaped(t)
	if err := project.ValidateSource(ref); err != nil {
		t.Errorf("project.ValidateSource of intact custody = %v, want nil", err)
	}
	if err := project.ValidateSource(ref.Archived()); err == nil {
		t.Errorf("project.ValidateSource of an archive that does not exist must fail")
	}
	path := ref.Path

	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	wantCustodyStop(t, "project.ValidateSource of a 0644 receipt", project.ValidateSource(ref))
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := project.ValidateSource(ref); err != nil {
		t.Errorf("project.ValidateSource after restoring the mode = %v", err)
	}

	if err := os.Chmod(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	wantCustodyStop(t, "project.ValidateSource of a 0755 receipt folder", project.ValidateSource(ref))
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(slices.Clone(orig), ' '), 0o600); err != nil {
		t.Fatal(err)
	}
	err := project.ValidateSource(ref)
	wantCustodyStop(t, "project.ValidateSource of changed bytes", err)
	if se, _ := sys.AsError(err); se != nil && !strings.Contains(se.Detail, ref.SHA256) {
		t.Errorf("project.ValidateSource names %q, want the expected SHA-256 %s", se.Detail, ref.SHA256)
	}
	if err := os.WriteFile(path, orig, 0o600); err != nil {
		t.Fatal(err)
	}

	// A symbolic link is not the regular file the digest was taken from.
	moved := path + ".moved"
	if err := os.Rename(path, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, path); err != nil {
		t.Fatal(err)
	}
	wantCustodyStop(t, "project.ValidateSource of a symlinked receipt", project.ValidateSource(ref))
	_ = dir
}

func TestSourceMessageAndCard(t *testing.T) {
	if got, want := project.SourceMessage(nil), "Source provenance: unavailable (legacy)"; got != want {
		t.Errorf("project.SourceMessage(nil) = %q, want %q", got, want)
	}
	if got, err := project.SourceCard(nil); err != nil || got != "Source provenance: unavailable (legacy)" {
		t.Errorf("project.SourceCard(nil) = %q, %v, want the legacy notice", got, err)
	}
	_, _, ref := shaped(t)
	message := "Diagnostic source provenance (not authority): " + ref.Path +
		"\nManifest sha256: " + ref.SHA256 + "; contract: " + ref.ContractSHA256 +
		". Retrieve full originals with read_file. Only the approved Intent authorizes implementation; omissions require ordinary Refine and new exact approval. Mapping is an aid, not proof."
	if got := project.SourceMessage(ref); got != message {
		t.Errorf("project.SourceMessage =\n%q\nwant\n%q", got, message)
	}
	card, err := project.SourceCard(ref)
	if err != nil || card != message+"\nSource comparison:\nThe request maps to the widget." {
		t.Errorf("project.SourceCard = %q, %v, want the message and the receipt's Source comparison (never the request)", card, err)
	}
	if strings.Contains(card, "Make widgets") {
		t.Errorf("project.SourceCard shows the raw request: %q", card)
	}
	if err := os.Chmod(ref.Path, 0o644); err != nil {
		t.Fatal(err)
	}
	card, err = project.SourceCard(ref)
	wantCustodyStop(t, "project.SourceCard of a receipt whose custody changed", err)
	if card != "" {
		t.Errorf("project.SourceCard returned %q with an error", card)
	}
}

func TestSourceRefValues(t *testing.T) {
	var none *project.SourceRef
	if none.Map() != nil || none.Archived() != nil {
		t.Errorf("project.SourceRef.Map and Archived of nil = %v, %v, want nil, nil (a legacy Intent)", none.Map(), none.Archived())
	}
	if !none.Equal(nil) || none.Equal(&project.SourceRef{}) || (&project.SourceRef{}).Equal(none) {
		t.Errorf("project.SourceRef.Equal: nil is equal only to nil")
	}
	a := &project.SourceRef{Path: "/r/current.json", SHA256: "aa", ContractSHA256: "cc"}
	same := &project.SourceRef{Path: "/r/current.json", SHA256: "aa", ContractSHA256: "cc"}
	if !a.Equal(same) || a.Equal(&project.SourceRef{Path: "/r/current.json", SHA256: "aa"}) || a.Equal(&project.SourceRef{Path: "/other", SHA256: "aa", ContractSHA256: "cc"}) {
		t.Errorf("project.SourceRef.Equal must compare the path, the digest and the contract")
	}
	if got := a.Map(); !equalMap(got, map[string]any{"path": "/r/current.json", "sha256": "aa", "contract_sha256": "cc"}) {
		t.Errorf("project.SourceRef.Map = %v, want path, sha256 and contract_sha256", got)
	}
	if got := (&project.SourceRef{Path: "/p", SHA256: "aa"}).Map(); !equalMap(got, map[string]any{"path": "/p", "sha256": "aa"}) {
		t.Errorf("project.SourceRef.Map without a contract = %v, want only path and sha256", got)
	}
	archived := a.Archived()
	if archived == a || archived.Path != filepath.Join("/r", "aa.json") || archived.SHA256 != "aa" || archived.ContractSHA256 != "cc" || a.Path != "/r/current.json" {
		t.Errorf("project.SourceRef.Archived = %+v, want the hash-named file beside the receipt, leaving the receipt's ref as it is", archived)
	}
}

func equalMap(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestRefreshSource(t *testing.T) {
	if got, err := project.RefreshSource(t.TempDir(), slug, "any", nil); got != nil || err != nil {
		t.Errorf("project.RefreshSource of a legacy Intent = %v, %v, want nil, nil", got, err)
	}
	dir, _, ref := shaped(t)
	before, _ := os.ReadFile(ref.Path)

	// A receipt that already binds the contract is returned unchanged.
	same, err := project.RefreshSource(dir, slug, ref.ContractSHA256, ref)
	if err != nil || !same.Equal(ref) {
		t.Errorf("project.RefreshSource to the bound contract = %v, %v, want %v unchanged", same, err, ref)
	}
	if now, _ := os.ReadFile(ref.Path); string(now) != string(before) {
		t.Errorf("project.RefreshSource rewrote a receipt that already binds the contract")
	}

	// Another contract: the receipt moves, the predecessor stays under its hash.
	newSHA := strings.Repeat("f", 64)
	moved, err := project.RefreshSource(dir, slug, newSHA, ref)
	if err != nil {
		t.Fatalf("project.RefreshSource: %v", err)
	}
	if moved.Path != ref.Path || moved.ContractSHA256 != newSHA || moved.SHA256 == ref.SHA256 {
		t.Errorf("project.RefreshSource = %+v, want the same path bound to contract %s with new bytes", moved, newSHA)
	}
	if err := project.ValidateSource(moved); err != nil {
		t.Errorf("project.RefreshSource left custody broken: %v", err)
	}
	if kept, err := os.ReadFile(ref.Archived().Path); err != nil || string(kept) != string(before) {
		t.Errorf("project.RefreshSource must keep the predecessor at %s byte for byte: %v", ref.Archived().Path, err)
	}
	if err := project.ValidateSource(ref.Archived()); err != nil {
		t.Errorf("project.SourceRef.Archived of the predecessor is not intact: %v", err)
	}
	doc := receipt(t, dir)
	prev, _ := doc["previous_receipt"].(map[string]any)
	if !equalMap(prev, ref.Archived().Map()) || doc["prior_contract_sha256"] != ref.ContractSHA256 || doc["contract_sha256"] != newSHA {
		t.Errorf("project.RefreshSource receipt = previous %v, prior %v, contract %v, want the archive %v, %s and %s", prev, doc["prior_contract_sha256"], doc["contract_sha256"], ref.Archived().Map(), ref.ContractSHA256, newSHA)
	}
	req, _ := doc["request"].(map[string]any)
	if req["text"] != "Make widgets.\n" || doc["comparison"] != "The request maps to the widget." {
		t.Errorf("project.RefreshSource changed the request or comparison: %v, %v", req, doc["comparison"])
	}

	// The caller's receipt must still be the current one.
	_, err = project.RefreshSource(dir, slug, strings.Repeat("e", 64), ref)
	wantCustodyStop(t, "project.RefreshSource with a receipt that is no longer current", err)
}

func TestRefineSource(t *testing.T) {
	if got, err := project.RefineSource(t.TempDir(), slug, project.NewSourceRequest("x", ""), "prior", nil); got != nil || err != nil {
		t.Errorf("project.RefineSource of a legacy Intent = %v, %v, want nil, nil", got, err)
	}
	dir, cfg, ref := shaped(t)
	before, _ := os.ReadFile(ref.Path)
	write(t, dir, project.IntentMDPath(slug), intentText("The correction maps to the widget."))
	write(t, dir, project.FeaturePath(slug), featureText+"\n")

	correction := project.NewSourceRequest("Make them blue.\n", "correction.txt")
	refined, err := project.RefineSource(dir, slug, correction, ref.ContractSHA256, ref)
	if err != nil {
		t.Fatalf("project.RefineSource: %v", err)
	}
	baseline, _ := project.Baseline(dir, slug, cfg)
	contract := project.Hash(dir, baseline)
	if refined.Path != ref.Path || refined.ContractSHA256 != contract || contract == ref.ContractSHA256 {
		t.Errorf("project.RefineSource = %+v, want the current receipt bound to the refined contract %s", refined, contract)
	}
	if err := project.ValidateSource(refined); err != nil {
		t.Errorf("project.RefineSource left custody broken: %v", err)
	}
	if bound, err := project.BindSource(dir, slug); err != nil || !bound.Equal(refined) {
		t.Errorf("project.BindSource after project.RefineSource = %v, %v, want %v", bound, err, refined)
	}
	doc := receipt(t, dir)
	corrections, _ := doc["corrections"].([]any)
	if len(corrections) != 1 {
		t.Fatalf("project.RefineSource corrections = %v, want the one correction", doc["corrections"])
	}
	c := corrections[0].(map[string]any)
	if c["text"] != "Make them blue.\n" || c["source_path"] != "correction.txt" || c["sha256"] != correction.SHA256 || c["prior_contract_sha256"] != ref.ContractSHA256 {
		t.Errorf("project.RefineSource correction = %v, want the text, path, digest and the contract it corrected", c)
	}
	if doc["comparison"] != "The correction maps to the widget." || doc["contract_sha256"] != contract || doc["prior_contract_sha256"] != ref.ContractSHA256 {
		t.Errorf("project.RefineSource receipt = comparison %v contract %v prior %v, want the refined comparison and contract", doc["comparison"], doc["contract_sha256"], doc["prior_contract_sha256"])
	}
	if prev, _ := doc["previous_receipt"].(map[string]any); !equalMap(prev, ref.Archived().Map()) {
		t.Errorf("project.RefineSource previous_receipt = %v, want %v", prev, ref.Archived().Map())
	}
	if kept, err := os.ReadFile(ref.Archived().Path); err != nil || string(kept) != string(before) {
		t.Errorf("project.RefineSource must keep the old receipt under its hash: %v", err)
	}
	if req, _ := doc["request"].(map[string]any); req["text"] != "Make widgets.\n" {
		t.Errorf("project.RefineSource changed the original request: %v", req)
	}

	// The comparison of the refined Intent must exist, and the old binding must still hold.
	write(t, dir, project.IntentMDPath(slug), "# T\n")
	if _, err := project.RefineSource(dir, slug, correction, "p", refined); err == nil {
		t.Errorf("project.RefineSource without a Source comparison must fail")
	}
	_, err = project.RefineSource(dir, slug, correction, "p", ref)
	wantCustodyStop(t, "project.RefineSource with a receipt that is no longer current", err)
}

func TestResolveApproval(t *testing.T) {
	dir, cfg, ref := shaped(t)
	if got, err := project.ResolveApproval(dir, slug, nil); got != nil || err != nil {
		t.Errorf("project.ResolveApproval(nil) = %v, %v, want nil, nil", got, err)
	}
	if got, err := project.ResolveApproval(dir, slug, &project.Approval{Slug: slug, SHA256: "x"}); got != nil || err != nil {
		t.Errorf("project.ResolveApproval of a legacy marker = %v, %v, want nil, nil", got, err)
	}
	baseline, _ := project.Baseline(dir, slug, cfg)
	oldHash := project.Hash(dir, baseline)
	marker := &project.Approval{Slug: slug, SHA256: oldHash, SourceProvenance: ref.Map()}

	got, err := project.ResolveApproval(dir, slug, marker)
	if err != nil || !got.Equal(ref) {
		t.Errorf("project.ResolveApproval of a current marker = %v, %v, want %v", got, err, ref)
	}

	// The marker of a contract that changed since: a refresh moved the receipt, and the marker
	// is resolved by its pinned digest in the history beside it.
	write(t, dir, project.FeaturePath(slug), featureText+"\n")
	baseline, _ = project.Baseline(dir, slug, cfg)
	if _, err := project.RefreshSource(dir, slug, project.Hash(dir, baseline), ref); err != nil {
		t.Fatal(err)
	}
	got, err = project.ResolveApproval(dir, slug, marker)
	if err != nil || !got.Equal(ref.Archived()) {
		t.Errorf("project.ResolveApproval of a stale marker = %v, %v, want the archive %v", got, err, ref.Archived())
	}

	// A marker that points elsewhere, or without a digest, is refused.
	for name, m := range map[string]map[string]any{
		"other path": {"path": filepath.Join(dir, "elsewhere.json"), "sha256": ref.SHA256},
		"bad digest": {"path": ref.Path, "sha256": "short"},
		"no pointer": {},
	} {
		_, err := project.ResolveApproval(dir, slug, &project.Approval{Slug: slug, SHA256: oldHash, SourceProvenance: m})
		wantCustodyStop(t, "project.ResolveApproval of a marker with "+name, err)
	}

	// An approval of the current contract needs the pointer to validate as it stands.
	current, err := project.BindSource(dir, slug)
	if err != nil {
		t.Fatal(err)
	}
	approved := &project.Approval{Slug: slug, SHA256: project.Hash(dir, baseline), SourceProvenance: current.Map()}
	if got, err := project.ResolveApproval(dir, slug, approved); err != nil || !got.Equal(current) {
		t.Errorf("project.ResolveApproval of the current approval = %v, %v, want %v", got, err, current)
	}
	if err := os.Chmod(current.Path, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = project.ResolveApproval(dir, slug, approved)
	wantCustodyStop(t, "project.ResolveApproval with broken custody", err)
}

func TestRestoreSource(t *testing.T) {
	if err := project.RestoreSource(nil, []byte("x")); err != nil {
		t.Errorf("project.RestoreSource(nil) = %v, want a no-op", err)
	}
	dir, _, ref := shaped(t)
	previous, _ := os.ReadFile(ref.Path)
	published, err := project.RefreshSource(dir, slug, strings.Repeat("a", 64), ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := project.RestoreSource(published, previous); err != nil {
		t.Fatalf("project.RestoreSource: %v", err)
	}
	if now, _ := os.ReadFile(ref.Path); string(now) != string(previous) {
		t.Errorf("project.RestoreSource did not put the previous bytes back")
	}
	if err := project.ValidateSource(ref); err != nil {
		t.Errorf("project.RestoreSource: the restored receipt fails its custody: %v", err)
	}
	if st, _ := os.Lstat(ref.Path); st.Mode().Perm() != 0o600 {
		t.Errorf("project.RestoreSource left mode %v, want 0600", st.Mode().Perm())
	}

	// A receipt someone else changed is never overwritten.
	published, err = project.RefreshSource(dir, slug, strings.Repeat("b", 64), ref)
	if err != nil {
		t.Fatal(err)
	}
	edited := []byte(`{"edited":"by someone"}` + "\n")
	if err := os.WriteFile(ref.Path, edited, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := project.RestoreSource(published, previous); err != nil {
		t.Errorf("project.RestoreSource of a concurrent edit = %v, want a quiet no-op", err)
	}
	if now, _ := os.ReadFile(ref.Path); string(now) != string(edited) {
		t.Errorf("project.RestoreSource overwrote a concurrent edit")
	}
}

func TestCheckoutAndPackageFacts(t *testing.T) {
	dir, cfg := newProject(t)
	facts, err := project.Checkout(dir)
	if err != nil {
		t.Fatalf("project.Checkout: %v", err)
	}
	physical, _ := filepath.EvalSymlinks(dir)
	gitDir, _ := filepath.EvalSymlinks(filepath.Join(dir, ".git"))
	if facts.Root != physical || facts.GitDir != gitDir || facts.ID != sha([]byte(gitDir)) {
		t.Errorf("project.Checkout = %+v, want the physical root %s, git dir %s and the SHA-256 of the git dir's path", facts, physical, gitDir)
	}
	if got, err := project.EncodeJSON(facts); err != nil || string(got) != `{"git_dir":"`+gitDir+`","id":"`+facts.ID+`","root":"`+physical+`"}` {
		t.Errorf("project.Checkout facts as JSON = %s, %v, want the root, git_dir and id keys", got, err)
	}
	// A linked worktree is its own checkout.
	linked := filepath.Join(t.TempDir(), "linked")
	runGit(t, dir, "worktree", "add", "-q", "-b", "other", linked)
	if other, err := project.Checkout(linked); err != nil || other.ID == facts.ID || other.GitDir == facts.GitDir {
		t.Errorf("project.Checkout of a linked worktree = %+v, %v, want its own git dir and id", other, err)
	}
	if _, err := project.Checkout(filepath.Join(dir, "missing")); err == nil {
		t.Errorf("project.Checkout of a missing folder must fail")
	}

	if err := project.WritePackageFacts(dir, slug, cfg); err != nil {
		t.Fatalf("project.WritePackageFacts: %v", err)
	}
	read := func() map[string]any {
		b, err := os.ReadFile(filepath.Join(dir, ".kogen/local/packages/widget.json"))
		if err != nil || !strings.HasSuffix(string(b), "\n") {
			t.Fatalf("project.WritePackageFacts file: %q, %v", b, err)
		}
		var doc map[string]any
		if err := json.Unmarshal(b, &doc); err != nil {
			t.Fatal(err)
		}
		return doc
	}
	baseline, _ := project.Baseline(dir, slug, cfg)
	doc := read()
	var paths []any
	for _, p := range slices.Sorted(slices.Values(baseline)) {
		paths = append(paths, p)
	}
	if doc["checkout_id"] != facts.ID || doc["sha256"] != project.Hash(dir, baseline) || doc["package_state"] != "valid" {
		t.Errorf("project.WritePackageFacts = %v, want checkout %s, hash %s, state valid", doc, facts.ID, project.Hash(dir, baseline))
	}
	if got, _ := doc["paths"].([]any); !slices.Equal(got, paths) {
		t.Errorf("project.WritePackageFacts paths = %v, want the sorted baseline %v", got, paths)
	}
	// An Intent that fails its check is recorded as invalid, and the file is replaced.
	write(t, dir, project.IntentMDPath(slug), "# A title that ends with a period.\n")
	if err := project.WritePackageFacts(dir, slug, cfg); err != nil {
		t.Fatal(err)
	}
	if doc := read(); doc["package_state"] != "invalid" {
		t.Errorf("project.WritePackageFacts of a broken Intent: package_state = %v, want invalid", doc["package_state"])
	}
	if _, err := os.Stat(filepath.Join(dir, ".kogen/local/packages/widget.json.tmp")); err == nil {
		t.Errorf("project.WritePackageFacts left its temporary file behind")
	}
}
