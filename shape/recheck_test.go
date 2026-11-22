package shape

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	rdhSlug    = "hash-me"
	rdhProject = "version: 1\ncommands:\n  checks:\n    - name: build\n      run: 'true'\n"
	rdhIntent  = "# Hash me\n\nSlug: hash-me\nModel: not applicable\nAcceptance:\n- .kogen/intents/hash-me/acceptance.feature\n\nDo a thing.\n"
	rdhFeature = "Feature: Hash me\n  Scenario: One\n    When a thing\n    Then it is done\n"
)

func rdhWrite(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// rdhRoot is a valid project with the draft of rdhSlug.
func rdhRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	rdhWrite(t, root, ".kogen/project.yaml", rdhProject)
	rdhWrite(t, root, ".kogen/intents/"+rdhSlug+"/intent.md", rdhIntent)
	rdhWrite(t, root, ".kogen/intents/"+rdhSlug+"/acceptance.feature", rdhFeature)
	return root
}

func TestRecheckDraftHashIsStableAndNonempty(t *testing.T) {
	parallel(t)
	root := rdhRoot(t)
	first, err := RecheckDraftHash(root, rdhSlug)
	if err != nil {
		t.Fatal(err)
	}
	if first == "" {
		t.Fatal("the hash of a valid draft is empty")
	}
	second, err := RecheckDraftHash(root, rdhSlug)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("two calls on an unchanged draft returned %q and %q", first, second)
	}
}

func TestRecheckDraftHashChangesWithDraftFile(t *testing.T) {
	parallel(t)
	root := rdhRoot(t)
	before, err := RecheckDraftHash(root, rdhSlug)
	if err != nil {
		t.Fatal(err)
	}
	rdhWrite(t, root, ".kogen/intents/"+rdhSlug+"/acceptance.feature", rdhFeature+"  Scenario: Two\n    When another thing\n    Then it is also done\n")
	after, err := RecheckDraftHash(root, rdhSlug)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatalf("changing acceptance.feature left the hash %q unchanged", before)
	}
}

func TestRecheckDraftHashErrorsWithoutValidProject(t *testing.T) {
	parallel(t)
	t.Run("missing draft: no project configuration", func(t *testing.T) {
		root := t.TempDir()
		if got, err := RecheckDraftHash(root, rdhSlug); err == nil {
			t.Fatalf("a draft in a root without .kogen/project.yaml returned hash %q and no error", got)
		}
	})
	t.Run("invalid draft: invalid project configuration", func(t *testing.T) {
		root := rdhRoot(t)
		rdhWrite(t, root, ".kogen/project.yaml", "version: [unterminated\n")
		if got, err := RecheckDraftHash(root, rdhSlug); err == nil {
			t.Fatalf("a draft in a root with an invalid .kogen/project.yaml returned hash %q and no error", got)
		}
	})
}
