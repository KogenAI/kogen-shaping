package shape

import (
	"strings"
	"testing"
)

func cacheKeys(reqs []map[string]any) []string {
	var keys []string
	for _, r := range reqs {
		k, _ := r["prompt_cache_key"].(string)
		keys = append(keys, k)
	}
	return keys
}

func TestAShapeConversationKeepsOneCacheIdentityForEveryRequest(t *testing.T) {
	parallel(t)
	f := newShpFx(t,
		shpAnswer(shpCall("c1", "submit", shpFiles())), // refused (uninspected): a second request follows
		shpSubmit("c2", shpFiles()),
		shpAck("c3"))
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	keys := cacheKeys(f.Prov.requests())
	if len(keys) != 3 || keys[0] == "" || len(keys[0]) > 64 || keys[0] != keys[1] || keys[1] != keys[2] {
		t.Fatalf("cache keys %q", keys)
	}
	events := shpNamed(f.events(), "cache_identity")
	var decisions []any
	var others []map[string]any
	for _, e := range events {
		if e["owner"] == "decision-check" {
			decisions = append(decisions, e["id"])
		} else {
			others = append(others, e)
		}
	}
	events = others
	if len(events) != 2 || events[0]["owner"] != "shape" || events[0]["id"] != keys[0] || events[1]["owner"] != "contract-review" || events[1]["id"] == keys[0] {
		t.Fatalf("saved identities %v, wire key %q", events, keys[0])
	}
	if checks := cacheKeys(f.Prov.contractRequests()); len(checks) != 1 || checks[0] != events[1]["id"] {
		t.Fatalf("the contract review wire key %q, saved %v", checks, events[1])
	}
	if checks := cacheKeys(f.Prov.decisionRequests()); len(checks) != 2 || len(decisions) != 2 || checks[0] != decisions[0] || checks[1] != decisions[1] || checks[0] == checks[1] || checks[0] == keys[0] {
		t.Fatalf("the decision check wire keys %q, saved %v", checks, decisions)
	}
	// A second invocation is another conversation.
	g := newShpFx(t, shpSubmit("c1", shpFiles()), shpAck("c2"))
	if _, err := g.shape("Add calc.is_even(n)."); err != nil {
		t.Fatal(err)
	}
	if other := cacheKeys(g.Prov.requests())[0]; other == keys[0] {
		t.Fatalf("two conversations share the identity %q", other)
	}
}

func TestAShapeConversationKeepsItsIdentityAcrossAWorkingWindowReset(t *testing.T) {
	parallel(t)
	f := newShpFx(t, readPadding(40)...)
	seedPadding(f)
	f.Prov.mu.Lock()
	f.Prov.seq = append(f.Prov.seq, shpSubmit("c1", shpFiles()), shpAck("c2"))
	f.Prov.mu.Unlock()
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	if len(shpNamed(f.events(), "working_context_reset")) == 0 {
		t.Fatal("the conversation never reset its window")
	}
	keys := cacheKeys(f.Prov.requests())
	for _, k := range keys {
		if k == "" || k != keys[0] {
			t.Fatalf("keys %q", keys)
		}
	}
}

func TestARefinementKeepsOneCacheIdentityOfItsOwn(t *testing.T) {
	parallel(t)
	f := newRefFx(t, refOpts{},
		refProbe("c1", "true"),
		refSubmit("c2", refIntentPath, refTitled))
	if exit, err := f.refine(f.hash0); err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	keys := cacheKeys(f.Prov.requests())
	if len(keys) != 2 || keys[0] == "" || keys[0] != keys[1] || strings.TrimSpace(keys[0]) != keys[0] {
		t.Fatalf("keys %q", keys)
	}
	events := 0
	for _, e := range f.evidenceEvents() {
		if e["event"] == "cache_identity" && e["owner"] != "contract-review" {
			events++
			if e["owner"] != "refine" || e["id"] != keys[0] {
				t.Fatalf("saved identity %v, wire key %q", e, keys[0])
			}
		}
	}
	if events != 1 {
		t.Fatalf("%d saved identities", events)
	}
}
