package shape

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShapeRestoresOnlyExactOwnedDraftHelpers(t *testing.T) {
	for _, mode := range []string{"exact", "changed checkout", "changed custody", "symlink custody", "symlink custody directory", "missing custody", "configured protection", "shared helper", "tracked custody"} {
		t.Run(mode, func(t *testing.T) {
			parallel(t)
			f := newShpFx(t,
				shpAnswer(shpCall("keep", ProposeFilesTool, shpFiles())),
				shpSubmit("first", shpFiles()), shpAck("again"), shpAck("last"))
			f.setReport("UNDEFINED", 1)
			if exit, err := f.shape("Add calc.is_even(n)."); err == nil || exit == 0 {
				t.Fatalf("first draft unexpectedly ready: %d %v", exit, err)
			}
			const step = "features/steps/is-even__steps.js"
			private := proposedDir(shpSlug) + step
			if !f.exists(private) {
				t.Fatal("accepted proposal custody was lost")
			}
			switch mode {
			case "changed checkout":
				f.write(step, f.read(step)+"\n// changed\n")
			case "changed custody":
				f.write(private, f.read(private)+"\n// changed\n")
			case "tracked custody":
				f.git("add", "-f", private)
				f.git("commit", "-qm", "Track copied custody")
			case "missing custody":
				if err := os.Remove(filepath.Join(f.Root, private)); err != nil {
					t.Fatal(err)
				}
			case "configured protection":
				f.write(".kogen/project.yaml", f.read(".kogen/project.yaml")+"\nprotect: ['features/steps/is-even__steps.js']\n")
			case "shared helper":
				f.write(".kogen/intents/other-one/intent.md", strings.Replace(f.read(".kogen/intents/is-even/intent.md"), "Slug: is-even", "Slug: other-one", 1))
			case "symlink custody directory":
				parent := filepath.Dir(filepath.Join(f.Root, private))
				if err := os.Rename(parent, parent+"-saved"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(parent+"-saved", parent); err != nil {
					t.Fatal(err)
				}
			case "symlink custody":
				path := filepath.Join(f.Root, private)
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(f.Root, step), path); err != nil {
					t.Fatal(err)
				}
			}
			partial := shpFiles()
			var files []any
			for _, raw := range partial["files"].([]any) {
				if raw.(map[string]any)["path"] != step {
					files = append(files, raw)
				}
			}
			partial["files"] = files
			f.setReport("FAILED", 1)
			f.Prov.mu.Lock()
			f.Prov.seq = append(f.Prov.seq, shpSubmit("retry", partial), shpAck("retry-ack"), shpAck("retry-last"))
			f.Prov.mu.Unlock()
			exit, err := f.shape("Add calc.is_even(n).")
			if mode == "exact" {
				if err != nil || exit != 0 {
					t.Fatalf("owned unchanged helper was refused on retry: %d %v\n%s", exit, err, f.Stderr)
				}
				restored := shpNamed(f.events(), "draft_helpers_restored")
				if len(restored) != 1 || !strings.Contains(f.read(step), "Then") {
					t.Fatalf("missing custody receipt: %v", restored)
				}
			} else {
				if err == nil || exit == 0 {
					t.Fatalf("unproven helper was adopted: %s", mode)
				}
				if len(shpNamed(f.events(), "draft_helpers_restored")) != 0 {
					t.Fatalf("unproven custody was restored: %s", mode)
				}
			}
		})
	}
}
