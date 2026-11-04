package consumes

import (
	"slices"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-verification/verify"
)

// Signatures and field types kogen-shaping's calls rely on.
var (
	_ func(*project.Intent) ([]verify.Declaration, error) = verify.Declarations
	_ string                                              = verify.Declaration{}.ID
	_ string                                              = verify.Declaration{}.Kind
	_ []string                                            = verify.Declaration{}.Observers
	_ []verify.DeclaredCheck                              = verify.Declaration{}.Checks
	_ string                                              = verify.DeclaredCheck{}.Name
	_ string                                              = verify.DeclaredCheck{}.Command
	_ int                                                 = verify.DeclaredCheck{}.Exit
	_ verify.Obligation                                   = verify.Obligation{}
)

const (
	capabilityBullet = `- evidence: {"id":"route","kind":"capability","inputs":["route.json"],"observers":[],"checks":[{"name":"support","command":"node route.cjs","exit":23,"output":"explicit rejection\n"}]}`
	observerBullet   = `- evidence: {"id":"obs","kind":"observer","inputs":["fixture.js"],"observers":["features/steps/obs.js"],"checks":[{"name":"good","command":"node obs.js good","exit":0,"output":"ok\n"},{"name":"bad","command":"node obs.js bad","exit":1,"output":"rejected\n"}]}`
)

func declarationsOf(feasibility ...string) ([]verify.Declaration, error) {
	return verify.Declarations(&project.Intent{
		Slug:        "demo",
		Acceptance:  []string{project.FeaturePath("demo"), "features/steps/obs.js"},
		Feasibility: feasibility,
	})
}

// verify.Declarations reads the `- evidence:` bullets of an Intent's Feasibility section, in
// order, ignoring the other bullets.
func TestDeclarations(t *testing.T) {
	got, err := declarationsOf("- none", capabilityBullet, "- prose bullet", observerBullet)
	if err != nil {
		t.Fatalf("verify.Declarations: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("verify.Declarations: want the 2 evidence bullets in order, got %+v", got)
	}
	route, obs := got[0], got[1]
	if route.ID != "route" || route.Kind != "capability" || len(route.Observers) != 0 ||
		!slices.Equal(route.Checks, []verify.DeclaredCheck{{Name: "support", Command: "node route.cjs", Exit: 23, Output: "explicit rejection\n"}}) {
		t.Errorf("verify.Declaration: the capability bullet was misread: %+v", route)
	}
	if obs.ID != "obs" || obs.Kind != "observer" || !slices.Equal(obs.Observers, []string{"features/steps/obs.js"}) || len(obs.Checks) != 2 ||
		obs.Checks[0].Name != "good" || obs.Checks[0].Exit != 0 || obs.Checks[1].Name != "bad" || obs.Checks[1].Exit != 1 || obs.Checks[1].Command != "node obs.js bad" {
		t.Errorf("verify.Declaration.Observers and Checks: the observer bullet was misread: %+v", obs)
	}

	if got, err := declarationsOf("- none"); err != nil || len(got) != 0 {
		t.Errorf("verify.Declarations: an Intent without evidence bullets has no declarations, got %+v %v", got, err)
	}
}

// verify.Declarations refuses a malformed declaration with intent/not_ready "invalid
// feasibility evidence" (exit 1) and one line naming the problem; kogen-shaping shows it.
func TestDeclarationsRefuseInvalidEvidence(t *testing.T) {
	cases := map[string]struct {
		bullets []string
		line    string
	}{
		"not JSON":         {[]string{"- evidence: {nope"}, "invalid evidence JSON"},
		"duplicate ids":    {[]string{capabilityBullet, capabilityBullet}, "evidence ids must be unique"},
		"unknown kind":     {[]string{`- evidence: {"id":"x","kind":"other","inputs":[],"observers":[],"checks":[]}`}, "invalid evidence declaration"},
		"no support":       {[]string{`- evidence: {"id":"x","kind":"capability","inputs":[],"observers":[],"checks":[]}`}, "x: invalid evidence checks or observers"},
		"outside observer": {[]string{`- evidence: {"id":"x","kind":"observer","inputs":[],"observers":["not/accepted.js"],"checks":[{"name":"good","command":"true","exit":0,"output":""},{"name":"bad","command":"false","exit":1,"output":""}]}`}, "x: invalid evidence checks or observers"},
		"bad polarity":     {[]string{`- evidence: {"id":"x","kind":"observer","inputs":[],"observers":["features/steps/obs.js"],"checks":[{"name":"good","command":"true","exit":1,"output":""},{"name":"bad","command":"false","exit":0,"output":""}]}`}, "x: observer good must expect exit 0 and bad a NONZERO integer exit"},
	}
	for name, c := range cases {
		_, err := declarationsOf(c.bullets...)
		e := sysError(t, err, "intent", "not_ready")
		if e.Detail != "invalid feasibility evidence" || e.Exit != 1 || !equalStrings(e.Lines, []string{c.line}) {
			t.Errorf("verify.Declarations: %s must refuse with exit 1 and the line %q, got exit %d %q %q", name, c.line, e.Exit, e.Detail, e.Lines)
		}
	}
}
