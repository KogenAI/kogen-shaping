package shape

import (
	"reflect"
	"strings"
	"testing"
)

// TestCheckItemsStateAllFieldsRequired checks the model-facing wording that every literal and
// decision check item carries every field, and that the schemas' shape is unchanged.
func TestCheckItemsStateAllFieldsRequired(t *testing.T) {
	parallel(t)
	itemOf := func(schema map[string]any) map[string]any {
		return schema["properties"].(map[string]any)["items"].(map[string]any)["items"].(map[string]any)
	}
	desc := func(item map[string]any, name string) string {
		p, _ := item["properties"].(map[string]any)[name].(map[string]any)
		d, _ := p["description"].(string)
		return d
	}
	has := func(what, text string, want ...string) {
		for _, w := range want {
			if !strings.Contains(text, w) {
				t.Errorf("%s lacks %q: %s", what, w, text)
			}
		}
	}
	lit, dec := itemOf(literalSchema()), itemOf(decisionSchema())

	has("literal instructions", literalInstructions, "carries every field, including status", "the empty string, never omitted",
		"for secret material leave literal, source_quote, package_value and why empty")
	has("decision instructions", decisionInstructions, "carries every field, including why", "the empty string", "never omitted")
	has("literal status description", desc(lit, "status"), "never omitted")
	has("literal why description", desc(lit, "why"), "never omitted", "empty string")
	has("decision why description", desc(dec, "why"), "never omitted", "empty string")

	if got, want := lit["required"], []any{"literal", "source", "source_quote", "package_value", "why", "status"}; !reflect.DeepEqual(got, want) {
		t.Errorf("literal required = %v", got)
	}
	if got, want := dec["required"], []any{"statement", "request_quotes", "verdict", "intent_value", "request_value", "difference", "why"}; !reflect.DeepEqual(got, want) {
		t.Errorf("decision required = %v", got)
	}
	if n := len(lit["properties"].(map[string]any)); n != 6 || lit["additionalProperties"] != false || literalToolDef()["strict"] != true {
		t.Errorf("literal item shape changed: %d properties", n)
	}
	if n := len(dec["properties"].(map[string]any)); n != 7 || dec["additionalProperties"] != false || decisionToolDef()["strict"] != true {
		t.Errorf("decision item shape changed: %d properties", n)
	}
}
