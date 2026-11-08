package shape

import "testing"

// A real recovery answered that controls were not prepared; forcing submit on
// the next inference made it generate an incomplete placeholder instead of using
// the preparation tools. Require progress without choosing the terminal tool.
func TestShapeToolFreePreparationKeepsToolsSelectable(t *testing.T) {
	parallel(t)
	for _, w := range []*forceWire{responsesWire, chatWire} {
		t.Run(w.name, func(t *testing.T) {
			parallel(t)
			f := w.newShape(t,
				w.readReadme("r1"),
				w.prose("Not ready: I must inspect the fixture and prepare controls before submitting."),
				w.readReadme("r2"),
				w.calls(shpCall("p1", ProposeFilesTool, shpFiles())),
				w.shapeSubmit("s1", shpFiles()), w.shapeAck("s2"))
			if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
				t.Fatalf("exit %d: %v", exit, err)
			}
			reqs := f.Prov.requests()
			if reqs[2]["tool_choice"] != "required" {
				t.Fatalf("preparation turn forces %v; want any available tool", reqs[2]["tool_choice"])
			}
			forceRetained(t, reqs, 2)
			if reqs[3]["tool_choice"] != w.auto {
				t.Fatal("preparation tool did not restore automatic choice")
			}
			shpEq(t, "published", f.read(refIntentPath), shpScopedIntentMD)
		})
	}
}
