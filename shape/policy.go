package shape

import (
	"embed"
	"fmt"
	"github.com/KogenAI/kogen-executor/agent"
	"github.com/KogenAI/kogen-loop/tools"
)

// Kogen's shaping tool names: the Shaper's and Refine's. The generic tools
// (read_file, view_image, list_files, search_text, documentation) are named and defined by
// github.com/KogenAI/kogen-loop/tools.
const (
	ProbeTool            = "probe"
	ProposeFilesTool     = "propose_files"
	SubmitTool           = "submit"
	SubmitRefinementTool = "submit_refinement"
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
	case ProbeTool:
		return probeDefinition(true)
	case ProposeFilesTool:
		return fun(name,
			"Retain complete owned Intent or step files without checking them. Each file supplies path and exactly one of content (whole text) or from_probe (a successful captured output handle from this invocation). Use captured output for programmatically generated controls.json or other large proof artifacts without retyping repository files. Same path/protection rules as submit. Proposals accumulate privately; malformed or unsafe mixed calls retain nothing. End with submit to check the complete package.",
			map[string]any{"files": proposalFileInputs()}, "files")
	case SubmitTool:
		return submitDef()
	}
	panic(fmt.Sprintf("shape: unknown tool %q", name))
}

func proposalFileInputs() map[string]any {
	l := pathAndContent(true)
	item := l["items"].(map[string]any)
	item["properties"].(map[string]any)["from_probe"] = str()
	item["required"] = []any{"path"}
	return l
}

func probeDefinition(capture bool) map[string]any {
	d := fun(ProbeTool,
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
	if capture {
		d["description"] = "Run a command with /bin/sh -c in a scratch copy of the working tree to investigate material uncertainty or prepare and check required proof artifacts. Accepted current proposals are applied to the copy first, then optional scratch files override them for this probe only. The project and retained proposals are untouched. Stopped after 5 minutes. Reports status, timeout, wall time and combined output. With capture_output=true, successful complete UTF-8 output is kept privately as a bounded artifact handle instead of being copied into the conversation. Generate a large controls.json programmatically, print only its contents, then use propose_files from_probe to retain it. Capturing confers no ownership, readiness or approval; normal proposal validation and every gate still apply."
		d["parameters"].(map[string]any)["properties"].(map[string]any)["capture_output"] = map[string]any{"type": "boolean"}
	}
	return d
}

func submitDef() map[string]any {
	return map[string]any{
		"type":   "function",
		"name":   SubmitTool,
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
				"decisions": map[string]any{
					"type":        "array",
					"description": "Optional, with questions or files, never with blocked: the technical, security, protocol, storage and limit decisions this shaping made, kept across its continuations. evidence lists repository files this shaping read and ran `<command>` items of probes that succeeded; constraints are the rules the decision imposes; uncertainty states what stays uncertain and is required without evidence; revises names a recorded decision this one replaces, with why it changed and evidence this round read or ran.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"decision":    str(),
							"evidence":    strList(),
							"constraints": strList(),
							"uncertainty": str(),
							"revises":     str(),
							"why":         str(),
						},
						"required": []any{"decision"},
					},
				},
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

// RefineToolDefinitions returns ordered model-facing Refine tools: the generic tools of
// kogen-loop, probe and submit_refinement. A name that is none of those panics: the names are
// fixed by the program.
func RefineToolDefinitions(names ...string) []map[string]any {
	out := make([]map[string]any, len(names))
	for i, name := range names {
		out[i] = refineToolDefinition(name)
	}
	return out
}

func refineToolDefinition(name string) map[string]any {
	if d, ok := tools.Definition(name); ok {
		return d
	}
	switch name {
	case ProbeTool:
		return probeDefinition(false)
	case SubmitRefinementTool:
		return map[string]any{
			"type":   "function",
			"name":   name,
			"strict": false,
			"description": "For edits, submit changed existing files; well-formed proposals accumulate in this run and omitted paths retain earlier proposed bytes. Submit original content only to intentionally revert an edit. " +
				shaperPromptFile("readiness-acknowledgment-guidance") + " Or report an explicit unsupported-work constraint.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"files":      pathAndContent(true),
					"constraint": str(),
				},
			},
		}
	}
	panic(fmt.Sprintf("shape: unknown tool %q", name))
}
