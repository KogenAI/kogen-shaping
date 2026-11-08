package shape

import (
	"strings"
	"testing"
)

func TestHelperPathRefusalExplainsOwnedDirectory(t *testing.T) {
	const bad = ".kogen/intents/is-even/driver.go"
	const good = "features/steps/is-even__driver/main.go"
	files := func(path string) map[string]any {
		pkg := shpFiles()
		list := pkg["files"].([]any)
		for _, raw := range list {
			f := raw.(map[string]any)
			if f["path"] == pfIntentPath {
				f["content"] = strings.Replace(f["content"].(string), "- features/steps/is-even__steps.js", "- features/steps/is-even__steps.js\n- "+path, 1)
			}
		}
		pkg["files"] = append(list, pfFile(path, "package main\n\nfunc main() {}\n"))
		return pkg
	}
	f := newShpFx(t, shpSubmit("bad", files(bad)), shpSubmit("fixed", files(good)))
	f.setReport("PASSED", 0)
	exit, err := f.shape("Add calc.is_even(n).")
	if err != nil || exit != 0 {
		t.Fatalf("exit %d: %v\nstderr: %s", exit, err, f.Stderr)
	}
	reqs := f.Prov.requests()
	shpContains(t, "initial guidance", shpBody(reqs[0]), "features/steps/is-even__driver/main.go")
	shpContains(t, "recovery", shpOutputs(reqs[1])["bad"], "put helper files in features/steps/is-even__<suffix>/")
	shpContains(t, "published helper declaration", f.read(pfIntentPath), good)
	refused := shpNamed(f.events(), "submission_refused")
	shpEq(t, "one refusal", len(refused), 1)
	shpEq(t, "ownership gate", refused[0]["gate"], any("step_ownership"))
}
