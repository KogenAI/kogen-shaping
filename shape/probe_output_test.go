package shape

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-loop/tools"
)

func captureOutputForTest(t *testing.T, sh *shaping, command string) string {
	t.Helper()
	out, err := sh.runTool(context.Background(), tools.Calls([]map[string]any{shpCall("capture", ProbeTool, map[string]any{"command": command, "capture_output": true})})[0])
	if err != nil {
		t.Fatal(err)
	}
	return out.Text
}

func TestCaptureLargeProbeOutputAsOwnedProposal(t *testing.T) {
	parallel(t)
	f, sh := newGuardedShaping(t)
	out := captureOutputForTest(t, sh, `printf '{"proof":"'; head -c 50000 /dev/zero | tr '\000' x; printf '"}'`)
	if len(out) > 2048 || !strings.Contains(out, "captured output: probe-output-1") {
		t.Fatalf("generated artifact still needs a large model reply: result bytes=%d", len(out))
	}
	p := ".kogen/intents/is-even/generated.json"
	text, err := sh.proposeFiles(map[string]any{"files": []any{map[string]any{"path": p, "from_probe": "probe-output-1"}}})
	if err != nil || !strings.HasPrefix(text, "retained:") {
		t.Fatalf("retention: %s %v", text, err)
	}
	want := `{"proof":"` + strings.Repeat("x", 50000) + `"}`
	if sh.proposed[p] != want {
		t.Fatal("captured artifact bytes changed")
	}
	if f.exists(p) {
		t.Fatal("capture or retention published the artifact before checks")
	}
	shpEq(t, "private proposal bytes", f.read(".kogen/local/shaping/is-even/proposed/"+p), want)
}

func TestCaptureGeneratedIntentThroughNormalShape(t *testing.T) {
	parallel(t)
	quoted := "'" + strings.ReplaceAll(shpScopedIntentMD, "'", "'\\''") + "'"
	files := shpFiles()["files"].([]any)
	proposal := append([]any{map[string]any{"path": refIntentPath, "from_probe": "probe-output-1"}}, files[1:]...)
	f := newShpFx(t,
		shpAnswer(shpCall("p", ProbeTool, map[string]any{"command": "printf '%s' " + quoted, "capture_output": true})),
		shpAnswer(shpCall("keep", ProposeFilesTool, map[string]any{"files": proposal})),
		shpSubmit("s", map[string]any{"files": []any{files[1]}}), shpAck("ack"))
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	shpEq(t, "checked published Intent", f.read(refIntentPath), shpScopedIntentMD)
	if len(shpNamed(f.events(), "probe_output_captured")) != 1 {
		t.Fatal("capture receipt missing")
	}
}

func TestProbeCaptureReferencesKeepProposalProtection(t *testing.T) {
	parallel(t)
	for _, name := range []string{"unknown", "ambiguous", "protected mixed", "failed probe", "invalid UTF-8", "changed", "symlink", "symlink parent"} {
		t.Run(name, func(t *testing.T) {
			parallel(t)
			f, sh := newGuardedShaping(t)
			command := "printf captured"
			if name == "failed probe" {
				command = "printf partial; exit 1"
			}
			if name == "invalid UTF-8" {
				command = `printf '\377'`
			}
			captureOutputForTest(t, sh, command)
			file := map[string]any{"path": ".kogen/intents/is-even/data.txt", "from_probe": "probe-output-1"}
			list := []any{file}
			switch name {
			case "unknown":
				file["from_probe"] = "unrelated-output"
			case "ambiguous":
				file["content"] = "other"
			case "protected mixed":
				list = append(list, map[string]any{"path": ".kogen/project.yaml", "from_probe": "probe-output-1"})
			case "changed", "symlink", "symlink parent":
				m, err := sh.writeManifest(Revision{})
				if err != nil {
					t.Fatal(err)
				}
				rows, ok := m["captured_probe_outputs"].([]any)
				if !ok || len(rows) != 1 {
					t.Fatal("capture navigation missing")
				}
				p := rows[0].(map[string]any)["source"].(string)
				if name == "changed" {
					os.WriteFile(p, []byte("tampered"), 0600)
				}
				if name == "symlink" {
					other := filepath.Join(t.TempDir(), "copy")
					os.WriteFile(other, []byte("captured"), 0600)
					os.Remove(p)
					os.Symlink(other, p)
				}
				if name == "symlink parent" {
					dir := filepath.Dir(p)
					other := filepath.Join(t.TempDir(), "copy")
					os.Rename(dir, other)
					os.Symlink(other, dir)
				}
			}
			text, err := sh.proposeFiles(map[string]any{"files": list})
			if err != nil || !strings.HasPrefix(text, "invalid proposal:") {
				t.Fatalf("unsafe capture accepted: %s %v", text, err)
			}
			if len(sh.proposed) != 0 || f.exists(".kogen/intents/is-even/data.txt") {
				t.Fatal("invalid mixed proposal retained or published files")
			}
		})
	}
}

func TestProbeCaptureIsBoundedAndRefineUnchanged(t *testing.T) {
	parallel(t)
	_, sh := newGuardedShaping(t)
	text, err := sh.captureProbeOutput(strings.Repeat("x", probeOutputMaxBytes+1))
	if err != nil || !strings.Contains(text, "capture refused") || len(sh.probeOutputs) != 0 {
		t.Fatal("oversized capture retained")
	}
	d := RefineToolDefinitions(ProbeTool)[0]
	props := d["parameters"].(map[string]any)["properties"].(map[string]any)
	if _, ok := props["capture_output"]; ok {
		t.Fatal("Refine advertises an unsupported capture operation")
	}
}
