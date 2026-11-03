package shape

import (
	"embed"
	"fmt"
	"github.com/KogenAI/kogen-executor/agent"
	"kogen-loop/tools"
)

//go:embed prompts/*.md
var shaperPromptFiles embed.FS

func shaperPromptFile(name string) string {
	b, err := shaperPromptFiles.ReadFile("prompts/" + name + ".md")
	if err != nil {
		panic(err)
	}
	return string(b)
}
func shaperPrompt(name string) string {
	return agent.Render(shaperPromptFile(name), map[string]string{
		"feasibility-guidance":              shaperPromptFile("feasibility-guidance"),
		"readiness-acknowledgment-guidance": shaperPromptFile("readiness-acknowledgment-guidance"),
	})
}

func fun(name, description string, props map[string]any, required ...string) map[string]any {
	req := make([]any, len(required))
	for i, r := range required {
		req[i] = r
	}
	return map[string]any{
		"type":        "function",
		"name":        name,
		"description": description,
		"parameters": map[string]any{
			"type":                 "object",
			"properties":           props,
			"required":             req,
			"additionalProperties": false,
		},
		"strict": false,
	}
}

func str() map[string]any { return map[string]any{"type": "string"} }
func strList() map[string]any {
	return map[string]any{"type": "array", "items": str()}
}

// pathAndContent is the {path, content} object of the file lists.
func pathAndContent(minItems bool) map[string]any {
	l := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type":       "object",
			"properties": map[string]any{"path": str(), "content": str()},
			"required":   []any{"path", "content"},
		},
	}
	if minItems {
		l["minItems"] = 1
	}
	return l
}

// ShaperToolDefinitions returns ordered model-facing shaping tools.
func ShaperToolDefinitions(names ...string) []map[string]any {
	out := make([]map[string]any, len(names))
	for i, name := range names {
		out[i] = shaperToolDefinition(name)
	}
	return out
}
func shaperToolDefinition(name string) map[string]any {
	if d, ok := tools.Definition(name); ok {
		return d
	}
	switch name {
	case agent.ProbeTool:
		return fun(name,
			"Run a quick throwaway command with /bin/sh -c in a scratch copy of the working tree (with its installed dependencies and builds), never in the project, to prove a risky assumption. Optional scratch files are written first. Stopped after 5 minutes. Returns the exit status, whether it timed out, the wall time and the combined output.",
			map[string]any{
				"command": str(),
				"files": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type":       "object",
						"properties": map[string]any{"path": str(), "content": str()},
						"required":   []any{"path", "content"},
					},
				},
			}, "command")
	case agent.ProposeFilesTool:
		return fun(name,
			"Retain Intent files without checking them yet, so a large package is written in bounded pieces across several replies: put one long file, or a few short ones, in each call and make one call per reply. Same path rules as submit's files; well-formed proposals accumulate (latest content for a path wins) and are kept privately under .kogen/local/shaping/<slug>/proposed/ as soon as they are accepted; a malformed or unsafe call retains nothing. A proposal is not a submission: it runs no checks and never counts as an invalid submission. End with submit, whose files (at least one, for example the last file) complete the package Kogen checks; a reply cut off before its tool call loses only that reply.",
			map[string]any{"files": pathAndContent(true)}, "files")
	case agent.SubmitTool:
		return submitDef()
	case agent.ShapeBatchTool:
		return map[string]any{
			"type":        "function",
			"name":        name,
			"description": "One blocking batch of 1–3 independent read/probe/draft jobs. Supply complete brief: revision {head,tree}, nonempty outcomes, interfaces, transitions, fixtures, exclusions, protected_paths array, artifacts [{path,expectation}]. Each job: unique id, role, exact scope, paths array, owns array. Read/probe own nothing; disjoint draft owns exact Intent/step files. Sequential is valid until all boundaries are known.",
			"parameters": map[string]any{
				"type":       "object",
				"properties": map[string]any{"brief": map[string]any{"type": "object"}, "jobs": map[string]any{"type": "array"}},
				"required":   []any{"brief", "jobs"},
			},
		}
	case agent.WorkerResultTool:
		return map[string]any{
			"type":        "function",
			"name":        name,
			"description": "Terminate with nonempty evidence and files [{path,content}]; read/probe files empty, draft only owns.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"evidence": str(),
					"files":    map[string]any{"type": "array"},
				},
				"required": []any{"evidence", "files"},
			},
		}
	}
	panic(fmt.Sprintf("shape: unknown tool %q", name))
}
func submitDef() map[string]any {
	return map[string]any{
		"type":   "function",
		"name":   agent.SubmitTool,
		"strict": false,
		"description": "Include nonempty ## Source comparison prose (at most 8192 UTF-8 bytes) comparing requested behaviors and settled non-secret literals against the actual complete contract, using addressed, omitted, conflicting, superseded with source/contract references. Do not expose secrets or raw private originals. Deliberate scope changes must be explicit; mappings are diagnostic aids, not semantic proof. End shaping with exactly one of questions, Intent files, or blocked. Within this run, well-formed file proposals accumulate: omitted paths retain earlier proposals; latest content wins. Submit only corrections on editing retries. " +
			shaperPromptFile("readiness-acknowledgment-guidance") + " A malformed or unsafe submission contributes no files. A blocked report is never authorization and cannot be paired with another result. Record a documentation lookup only when this conversation's documentation tool returned that status and reference. An explicit question kind other than product or handoff is rejected: decide technical questions inside shaping; only kind product or handoff may reach the caller.",
		"parameters": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"questions": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"question": str(),
							"kind": map[string]any{
								"type":        "string",
								"description": "Optional. Omit or product for a UI/UX or product question. handoff for a mandatory authentication or security handoff. Any other explicit kind is rejected.",
							},
							"options": map[string]any{
								"type": "array",
								"items": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"label":       str(),
										"recommended": map[string]any{"type": "boolean"},
									},
									"required": []any{"label"},
								},
							},
						},
						"required": []any{"question"},
					},
				},
				"files": pathAndContent(true),
				"blocked": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"paths":  strList(),
						"reason": str(),
					},
					"required": []any{"paths", "reason"},
				},
			},
		},
	}
}
