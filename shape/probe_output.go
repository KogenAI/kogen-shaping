package shape

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/KogenAI/kogen-loop/tools"
)

// A generated proof can contain several bounded implementation overlays. Keep
// its transport outside the model's reply, without enlarging the working window
// or the independent per-implementation controls limits.
const probeOutputMaxBytes = 2 * tools.InputBudgetBytes

type capturedProbeOutput struct {
	path, hash string
	bytes      int
}

func (sh *shaping) captureProbeOutput(output string) (string, error) {
	if len(output) > probeOutputMaxBytes || !utf8.ValidString(output) {
		return fmt.Sprintf("capture refused: output must be UTF-8 and at most %d bytes", probeOutputMaxBytes), nil
	}
	root, err := filepath.EvalSymlinks(sh.ev.Dir)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "probe-outputs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil || resolved != dir {
		return "capture refused: private output directory passes through a symlink", nil
	}
	f, err := os.CreateTemp(dir, "output-")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, err = f.WriteString(output); err == nil {
		err = f.Close()
	} else {
		f.Close()
	}
	if err != nil {
		os.Remove(name)
		return "", err
	}
	if sh.probeOutputs == nil {
		sh.probeOutputs = map[string]capturedProbeOutput{}
	}
	id := fmt.Sprintf("probe-output-%d", len(sh.probeOutputs)+1)
	x := capturedProbeOutput{path: filepath.Join(sh.ev.Dir, "probe-outputs", filepath.Base(name)), hash: ContentHash([]byte(output)), bytes: len(output)}
	sh.probeOutputs[id] = x
	sh.ev.Record(map[string]any{"event": "probe_output_captured", "id": id, "sha256": x.hash, "bytes": x.bytes})
	return fmt.Sprintf("captured output: %s; sha256=%s; bytes=%d. Retain it with propose_files using {\"path\":\"<owned artifact path>\",\"from_probe\":\"%s\"}. No proposal or readiness was created.", id, x.hash, x.bytes, id), nil
}

func (sh *shaping) capturedOutputManifest() []any {
	var rows []any
	for _, id := range sortedKeys(sh.probeOutputs) {
		x := sh.probeOutputs[id]
		rows = append(rows, map[string]any{"id": id, "sha256": x.hash, "bytes": x.bytes, "source": x.path})
	}
	return rows
}

// References are current-invocation capabilities, not arbitrary file reads.
// Expansion precedes the ordinary transactional path/protection validation.
func (sh *shaping) expandProbeOutputs(list []any) ([]any, string) {
	expanded := make([]any, 0, len(list))
	for _, raw := range list {
		m, ok := raw.(map[string]any)
		if !ok {
			expanded = append(expanded, raw)
			continue
		}
		ref, hasRef := m["from_probe"]
		if !hasRef {
			expanded = append(expanded, raw)
			continue
		}
		if _, hasContent := m["content"]; hasContent {
			return nil, "choose exactly one of content or from_probe"
		}
		id, ok := ref.(string)
		x, known := sh.probeOutputs[id]
		if !ok || !known {
			return nil, "from_probe must name captured output from this invocation"
		}
		st, err := os.Lstat(x.path)
		if err != nil || !st.Mode().IsRegular() || st.Size() != int64(x.bytes) {
			return nil, "captured probe output is missing or changed: " + id
		}
		rel, err := filepath.Rel(sh.ev.Dir, x.path)
		if err != nil {
			return nil, "captured probe output is unavailable: " + id
		}
		data, safe := custodyBytes(sh.ev.Dir, filepath.ToSlash(rel))
		if !safe || ContentHash(data) != x.hash {
			return nil, "captured probe output is missing or changed: " + id
		}
		expanded = append(expanded, map[string]any{"path": m["path"], "content": string(data)})
	}
	return expanded, ""
}
