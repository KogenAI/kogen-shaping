package shape

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

// certification is what one readiness observation of the package on disk leaves: the feedback and
// outcome it gives, and, for a fresh check, its result and observations.
type certification struct {
	text         string
	out          outcome
	ready        *verify.Ready // the fresh check's result; nil when the checked observations were reused
	path         string        // the observations file, relative to the checkout
	observations string        // the observations of a failing result
	fresh        bool          // the observations were written
}

// gateEffects are the changes of shaping state that the readiness gates make while a readiness
// observation overlaps other checks. They are applied when the observation is used.
type gateEffects struct {
	unpassable    *completedUnpassable
	unpassableSet bool
	preparation   []preparationSignature
}

func (sh *shaping) applyGateEffects(fx *gateEffects) {
	if fx.unpassableSet {
		sh.unpassableResult = fx.unpassable
	}
	sh.preparationRefused = append(sh.preparationRefused, fx.preparation...)
}

// deferredEvidence is the evidence of an overlapped readiness observation: time and counts are
// accounted at once, events are held until the observation is used, so they are recorded in the
// fixed order of the checks and never in completion order.
type deferredEvidence struct {
	ev     *Evidence
	mu     sync.Mutex
	events []map[string]any
}

var _ verify.Evidence = (*deferredEvidence)(nil)

func (d *deferredEvidence) Count(kind string) { d.ev.Count(kind) }

func (d *deferredEvidence) Timed(phase string, fn func() error) error { return d.ev.Timed(phase, fn) }

func (d *deferredEvidence) Record(event map[string]any) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.events = append(d.events, event)
}

func (d *deferredEvidence) Probe(fn func() (verify.ProbeResult, error)) (verify.ProbeResult, error) {
	t := d.ev.Millis()
	r, err := fn()
	ended := d.ev.Millis()
	d.Record(probeEvent(t, ended, r, err))
	d.ev.Interval("probe", t, ended)
	return r, err
}

// flush records the held events in the order they happened.
func (d *deferredEvidence) flush() {
	d.mu.Lock()
	events := d.events
	d.events = nil
	d.mu.Unlock()
	for _, e := range events {
		d.ev.Record(e)
	}
}

// readinessRun is the readiness observation of a submitted package that runs together with the
// static checks and the model checks.
type readinessRun struct {
	cancel context.CancelFunc
	done   chan struct{}
	hash   string // the package the observation started on
	cert   certification
	err    error
	events *deferredEvidence
	fx     gateEffects
}

// stop abandons the observation and waits for it to end; it does nothing for a nil run.
func (r *readinessRun) stop() {
	if r == nil {
		return
	}
	r.cancel()
	<-r.done
}

// lockedWriter makes the session's progress lines safe to write from concurrent checks.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// startReadiness starts the readiness observation of the package on disk, with the separate
// working copy readiness always uses. Nothing it observes changes shaping state or the events
// until joinReadiness uses it.
func (sh *shaping) startReadiness(ctx context.Context, files []string, hash string) *readinessRun {
	if _, ok := sh.s.Stderr.(*lockedWriter); !ok {
		sh.s.Stderr = &lockedWriter{w: sh.s.Stderr}
	}
	ctx, cancel := context.WithCancel(ctx)
	run := &readinessRun{cancel: cancel, done: make(chan struct{}), hash: hash, events: &deferredEvidence{ev: sh.ev}}
	// Computing the protected files writes the configuration's diagnostics: the observation gets
	// its own copy, as it runs while the shaping's package hash is computed.
	cfg := *sh.cfg
	rd := *sh.rd
	rd.Config, rd.Evidence = &cfg, run.events
	rd.Gate = func(ready *verify.Ready) error {
		if err := readinessGate(ready); err != nil {
			return err
		}
		if err := sh.preparationGateInto(ready, &run.fx); err != nil {
			return err
		}
		return sh.unpassableGateInto(ctx, ready, run.events.Record, &run.fx)
	}
	go func() {
		defer close(run.done)
		run.cert, run.err = sh.certify(ctx, files, &rd, &run.fx)
	}()
	return run
}

// joinReadiness is the readiness observation of the package on disk, whose hash is given. The
// overlapped observation is used, with its held events and gate effects, when it ran on that
// package; otherwise readiness runs on it now.
func (sh *shaping) joinReadiness(ctx context.Context, files []string, run *readinessRun, hash string) (certification, error) {
	if run != nil {
		<-run.done
		if run.hash == hash {
			run.events.flush()
			sh.applyGateEffects(&run.fx)
			return run.cert, run.err
		}
	}
	return sh.certify(ctx, files, sh.rd, nil)
}

// applyCertification makes the observation shaping's current one.
func (sh *shaping) applyCertification(c certification) {
	sh.lastFresh, sh.lastObservations = false, ""
	if c.ready == nil {
		return
	}
	sh.ready, sh.lastObservationsPath = c.ready, c.path
	if c.fresh {
		sh.lastFresh = true
		sh.lastObservations = c.observations
	}
}

// certify observes readiness of the package on disk with rd. It changes no shaping state: the
// caller applies the result. With effects, the gates leave their state changes there.
func (sh *shaping) certify(ctx context.Context, files []string, rd *verify.Readiness, fx *gateEffects) (certification, error) {
	if _, err := project.SourceComparison(sh.root, sh.slug); err != nil {
		return certification{}, err
	}
	// Fixers may change submitted declarations. The final formatted package must retain the
	// same executable artifacts before it can be certified.
	if intent, _, err := project.CheckIntent(sh.root, sh.slug, nil); err == nil {
		if problems := sh.retainedProblems(intent, files); len(problems) > 0 {
			e := sys.Fail("intent", "lint", "the Intent needs changes", 1)
			e.Lines = problems
			return certification{}, e
		}
	}
	if sh.ready != nil && rd.Current(sh.s, sh.slug, sh.ready) {
		return certification{text: "accepted: unchanged checked observations acknowledged", out: outcome{kind: outReady}}, nil
	}
	var ready *verify.Ready
	err := sh.ev.Timed("readiness", func() error {
		var err error
		ready, err = rd.Check(ctx, sh.s, sh.slug)
		return err
	})
	if err != nil {
		return certification{}, err
	}
	c := certification{ready: ready}
	reference := filepath.Join(sh.root, ".kogen/local/shaping", sh.slug, fmt.Sprintf("observations-%d.txt", sh.requests))
	c.path = fmt.Sprintf(".kogen/local/shaping/%s/observations-%d.txt", sh.slug, sh.requests)
	observations := verify.Observations(sh.slug, ready, reference)
	if err := os.WriteFile(reference, []byte(observations), 0o666); err != nil {
		return c, err
	}
	c.fresh = true
	if ready.Failures() {
		c.observations = observations
		c.text, c.out = verify.Acknowledgment(sh.slug, ready, reference), outcome{kind: outObservations}
		return c, nil
	}
	c.text, c.out = "accepted: the Intent is ready", outcome{kind: outReady}
	return c, nil
}
