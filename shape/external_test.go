package shape

import (
	"slices"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
)

func TestSumUsagePartialWhenStatesDiffer(t *testing.T) {
	parallel(t)
	a, b := 1.5, 2.0
	one := executor.Usage{CostUSD: &a, State: executor.UsageComplete}
	two := executor.Usage{CostUSD: &b, State: executor.UsageUnknown}
	got := sumUsage(one, two, false)
	if got.State != executor.UsagePartial {
		t.Fatalf("state %q, want partial", got.State)
	}
	if got.CostUSD == nil || *got.CostUSD != 3.5 {
		t.Fatalf("cost %v, want 3.5", got.CostUSD)
	}
}

// TestOwnModelMSIsPerRound: Claude Code's duration_api_ms of a resumed session is cumulative, so
// summing the reported values counted earlier rounds again and left the shape time account with
// more model time than shaper time (a negative tools time).
func TestOwnModelMSIsPerRound(t *testing.T) {
	parallel(t)
	x := &extSession{}
	var got []int64
	for _, total := range []int64{27180, 37176, 43626, 10, 10} {
		own := x.ownModelMS(&total)
		got = append(got, *own)
	}
	want := []int64{27180, 9996, 6450, 10, 0}
	if !slices.Equal(got, want) {
		t.Fatalf("per-round model time = %v, want %v", got, want)
	}
	if x.ownModelMS(nil) != nil {
		t.Fatal("an unknown model time became known")
	}
}
