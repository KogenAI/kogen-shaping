package shape

import (
	"fmt"
	"strings"

	"github.com/KogenAI/kogen-loop/tools"
)

// decisionEvidence supplies the values behind citations, independently of the
// candidate's claims. Only observed tracked paths and retained probe outputs
// qualify; a marker by itself does not establish a repository fact.
func (sh *shaping) decisionEvidence(intent string) (string, error) {
	var blocks []string
	seen := map[string]bool{}
	for _, citation := range evidenceCitations(decisionIntent(intent)) {
		for _, raw := range splitCitation(citation) {
			item := parseEvidenceItem(raw)
			if item.empty {
				continue
			}
			if item.command != "" {
				key := "command:" + item.command
				if seen[key] {
					continue
				}
				seen[key] = true
				text, ok := sh.probeTexts[item.command]
				if sh.ext != nil || !ok {
					blocks = append(blocks, fmt.Sprintf("Probe %q: no output retained by this invocation", item.command))
					continue
				}
				blocks = append(blocks, fmt.Sprintf("Probe %q, observed output:\n%s", item.command, tools.Cut(text)))
				continue
			}
			path := strings.TrimSuffix(cleanCitedPath(item.path), "/")
			if seen[path] {
				continue
			}
			seen[path] = true
			tracked, err := sh.headFiles()
			if err != nil {
				return "", err
			}
			observed := sh.readPaths[path]
			if headDirectory(tracked, path) {
				observed = sh.listedCovers(path)
			}
			if sh.ext != nil {
				observed = sh.inspected
			}
			if !observed {
				blocks = append(blocks, fmt.Sprintf("Path %q: not observed by this shaping", path))
				continue
			}
			// HEAD is the preserved baseline. Reading its Git object avoids following a
			// checkout symlink or mistaking a candidate's assertion for source bytes.
			body, ok := sh.repo.Show("HEAD", strings.TrimSuffix(path, "/"))
			if !ok {
				blocks = append(blocks, fmt.Sprintf("Path %q: not a tracked baseline object", path))
				continue
			}
			blocks = append(blocks, fmt.Sprintf("Path %q, baseline HEAD bytes independently read by this check:\n%s", path, tools.Cut(body)))
		}
	}
	if len(blocks) == 0 {
		return "", nil
	}
	return "\n\n# Observed repository evidence\n\n" + strings.Join(blocks, "\n\n"), nil
}
