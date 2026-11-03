'use strict';
module.exports = `package shape
import (
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func TestKTitleRetained(t *testing.T) {
 const long = "Add an is-even check for integers and also reject every string"
 const short = "Add an is-even check for integers and also reject"
 input, expected, events := long, short, 1
 if os.Getenv("K_TITLE_CONTROL") != "" { input, events = short, 0 }
 if os.Getenv("K_TITLE_CONTROL") == "bad" { expected = "Wrong retained title" }
 proposal := shpTitled(input, "- Non-integers raise TypeError.")
 f := newShpFx(t, shpSubmit("c1", proposal), shpSubmit("c2", proposal), shpSubmit("c3", proposal))
 f.setReport("PASSED", 0)
 _, err := f.shape("Add calc.is_even(n).")
 shpWantError(t, err, "shape", "invalid_output", "", 1)
 shapeAbsent(t, f)
 retained := f.read(filepath.Join(".kogen/local/shaping/is-even/proposed", ".kogen/intents/is-even/intent.md"))
 shpEq(t, "retained shortened title", strings.SplitN(retained, "\\n", 2)[0], "# "+expected)
 // Preserve every byte other than the title.
 original := proposal["files"].([]any)[0].(map[string]any)["content"].(string)
 shpEq(t, "non-title content", strings.SplitN(retained, "\\n", 2)[1], strings.SplitN(original, "\\n", 2)[1])
 shpEq(t, "shortening events", len(shpNamed(f.events(), "title_shortened")), events)
 shpEq(t, "refusals", strings.Join(shpRefused(f), " "), "1:intent 2:intent 3:intent")
}
`;
