package shape

import (
	"testing"
	"time"
)

// TestExternalBriefStatesTheBudget: the external session brief carries the budget it is given.
func TestExternalBriefStatesTheBudget(t *testing.T) {
	parallel(t)
	if externalBriefFor(25*time.Minute) == externalBriefFor(shapingBudget) {
		t.Fatal("the external brief does not depend on the budget")
	}
	shpEq(t, "default brief", externalBrief(), externalBriefFor(shapingBudget))
	shpContains(t, "25 brief", externalBriefFor(25*time.Minute), "25")
}
