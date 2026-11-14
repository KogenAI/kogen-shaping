package shape

import (
	"fmt"
	"sort"
	"testing"
)

// strictSchemaProblems walks a tool parameter schema and returns every object schema, at any depth,
// that declares a property missing from its required list or lacks additionalProperties false: the
// rules OpenAI enforces on strict tools.
func strictSchemaProblems(path string, schema any) []string {
	m, ok := schema.(map[string]any)
	if !ok {
		return nil
	}
	var out []string
	if props, ok := m["properties"].(map[string]any); ok {
		required := map[string]bool{}
		if list, ok := m["required"].([]any); ok {
			for _, r := range list {
				if s, ok := r.(string); ok {
					required[s] = true
				}
			}
		}
		names := make([]string, 0, len(props))
		for name := range props {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if !required[name] {
				out = append(out, fmt.Sprintf("%s: property %q is declared but not in required", path, name))
			}
			out = append(out, strictSchemaProblems(path+"."+name, props[name])...)
		}
		if m["additionalProperties"] != false {
			out = append(out, path+": additionalProperties is not false")
		}
	}
	if items, ok := m["items"]; ok {
		out = append(out, strictSchemaProblems(path+"[]", items)...)
	}
	for _, key := range []string{"anyOf", "oneOf", "allOf"} {
		if list, ok := m[key].([]any); ok {
			for i, s := range list {
				out = append(out, strictSchemaProblems(fmt.Sprintf("%s.%s[%d]", path, key, i), s)...)
			}
		}
	}
	return out
}

// TestStrictToolSchemasAreValid checks the parameter schema of every tool Shape defines with
// strict true. A new strict tool must be added to the list.
func TestStrictToolSchemasAreValid(t *testing.T) {
	tools := map[string]map[string]any{
		"literal":  literalToolDef(),
		"question": questionToolDef(),
		"decision": decisionToolDef(),
		"review":   contractReviewToolDef(),
		"confirm":  contractConfirmationToolDef(),
		"example":  {"strict": true, "parameters": exampleSchema()},
	}
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if tools[name]["strict"] != true {
			t.Errorf("%s: tool is not defined with strict true", name)
		}
		for _, problem := range strictSchemaProblems(name, tools[name]["parameters"]) {
			t.Error(problem)
		}
	}
}
