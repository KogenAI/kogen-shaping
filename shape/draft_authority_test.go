package shape

import (
	"strings"
	"testing"
)

// Recovery needs every draft byte for repair without promoting an earlier
// model's additional scope to caller authority. Other contracts remain frozen.
func TestContextKeepsUnapprovedDraftWithoutScopeAuthority(t *testing.T) {
	parallel(t)
	for _, latched := range []bool{false, true} {
		t.Run(map[bool]string{false: "existing contract", true: "unapproved recovery"}[latched], func(t *testing.T) {
			parallel(t)
			f, sh := newGuardedShaping(t)
			for _, raw := range shpFiles()["files"].([]any) {
				x := raw.(map[string]any)
				f.write(x["path"].(string), x["content"].(string))
			}
			own := ".kogen/intents/is-even/intent.md"
			other := ".kogen/intents/other-one/intent.md"
			f.write(other, strings.Replace(f.read(own), "Slug: is-even", "Slug: other-one", 1))
			if latched {
				f.write(".kogen/local/shaping/is-even/not-ready.json", `{"gate":"decision_check","problems":["extra scope in the draft"]}`)
			}
			_, index, err := BuildContextIndex(f.Repo, sh.cfg, shpSlug, nil, ContextOptions{FullPaths: []string{own, other, "README.md"}})
			if err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for _, item := range index.Measure.Selection {
				if item.Path != own && item.Path != other && item.Path != "README.md" {
					continue
				}
				seen[item.Path] = true
				if item.IncludedBodyBytes != len(f.read(item.Path)) || item.OmittedBodyBytes != 0 {
					t.Fatalf("contract body clipped: %+v", item)
				}
				if latched && item.Path == own {
					if !strings.Contains(item.Reason, "unapproved") || strings.Contains(item.Reason, "authoritative") {
						t.Fatalf("an unapproved model draft became scope authority: %+v", item)
					}
				} else if !strings.Contains(item.Reason, "authoritative") {
					t.Fatalf("caller convention or another contract lost authority: %+v", item)
				}
			}
			if len(seen) != 3 {
				t.Fatalf("missing complete context bodies: %v", seen)
			}
		})
	}
}
