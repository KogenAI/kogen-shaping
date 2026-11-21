package shape

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

// The verdicts of Recheck, in the words of the request.
const (
	RecheckRunnerFailure = "runner failure"
	RecheckReferenceFail = "reference check failed"
	RecheckBadDraft      = "bad draft"
)

// The RunnerKind of a runner failure, in words.
const (
	RunnerTimeout          = "timeout"
	RunnerMissingTool      = "missing tool"
	RunnerKilledCommand    = "killed command"
	RunnerIncompleteReport = "incomplete report"
	RunnerSetupError       = "setup error"
)

// setupFailureClass is the class of a failed setup command, as Shape records it.
const setupFailureClass = "setup_failure"

// recheckKept is the Kind of a retained contract-review property that asserts unchanged behaviour.
const recheckKept = "kept"

// The gates a recheck can find without the evidence it needs.
const (
	recheckGateReview   = "contract_review"
	recheckGateDraft    = "draft"
	recheckGateControls = "wrong_implementation"
)

// RecheckProperty is one property of the retained contract review: its name, its kind (`kept` for
// a property that asserts unchanged behaviour) and the names of the scenarios that declare it.
type RecheckProperty struct {
	Name      string
	Kind      string
	Scenarios []string
}

// RecheckInput is what Recheck diagnoses.
type RecheckInput struct {
	// Project is the runner checkout that holds the retained draft; Pinned is a pinned
	// checkout without the draft. Neither is written.
	Project, Pinned string
	Slug            string
	// DraftHash is the package hash of the retained draft that the caller recorded; see
	// RecheckDraftHash.
	DraftHash string
	// Properties are the properties of the retained contract review.
	Properties []RecheckProperty
	// Timeout is the limit of one acceptance run; zero is the readiness limit.
	Timeout time.Duration
}

// RecheckReport is the finding of Recheck. Verdict is one of RecheckRunnerFailure,
// RecheckReferenceFail and RecheckBadDraft, or empty when the evidence supports none of them:
// AttributionOpen then says the same failure appears on both trees, and Unavailable that the
// evidence the diagnostic needs is missing (Gate names the gate). ExitCode is the exit code of the
// run the finding is about, or -1 for a run that timed out.
type RecheckReport struct {
	Verdict         string
	Scenario        string
	Command         string
	ExitCode        int
	Class           string
	RunnerKind      string
	AttributionOpen bool
	Unavailable     bool
	Gate            string
	Evidence        []string
	Text            string
}

// RecheckDraftHash is the package hash of the draft of slug in root, the hash approve prints.
func RecheckDraftHash(root, slug string) (string, error) {
	cfg, err := project.LoadConfig(root)
	if err != nil {
		return "", err
	}
	baseline, err := project.Baseline(root, slug, cfg)
	if err != nil {
		return "", err
	}
	return project.Hash(root, baseline), nil
}

func recheckUnavailable(gate string, evidence ...string) RecheckReport {
	return RecheckReport{Unavailable: true, Gate: gate, Evidence: evidence,
		Text: fmt.Sprintf("Recheck is unavailable: the %s gate has no retained evidence to check against; nothing was run and nothing is inferred. %s", gate, strings.Join(evidence, " "))}
}

// Recheck names the cause of a refused check on a retained draft. It runs the draft's acceptance
// in scratch copies of Pinned, once on the unchanged tree and once with the correct implementation
// of the draft's controls.json, and makes no model request, no retry and no write to Project or
// Pinned. A run that fails for the runner's sake ends the diagnostic after that run.
func Recheck(ctx context.Context, in RecheckInput) (RecheckReport, error) {
	slug := in.Slug
	intent, err := project.ReadIntent(in.Project, slug, nil)
	feature, featureErr := os.ReadFile(filepath.Join(in.Project, project.FeaturePath(slug)))
	if err != nil || featureErr != nil {
		return recheckUnavailable(recheckGateDraft, fmt.Sprintf("the draft %s is missing or unreadable in %s.", slug, in.Project)), nil
	}
	current, err := RecheckDraftHash(in.Project, slug)
	if err != nil {
		return RecheckReport{}, err
	}
	switch {
	case in.DraftHash == "":
		return recheckUnavailable(recheckGateReview, fmt.Sprintf("no draft hash was recorded for the retained contract review; the current draft hash is %s.", current)), nil
	case in.DraftHash != current:
		return recheckUnavailable(recheckGateReview, fmt.Sprintf("the retained contract review was recorded for draft hash %s, but the current draft hash is %s.", in.DraftHash, current)), nil
	case len(in.Properties) == 0:
		return recheckUnavailable(recheckGateReview, "the retained contract review recorded no properties, so which scenarios are kept is unknown."), nil
	}
	scenarios, err := project.ParseGherkin(string(feature))
	if err != nil {
		return RecheckReport{}, err
	}
	cfg, err := project.LoadConfig(in.Pinned)
	if err != nil {
		return RecheckReport{}, err
	}
	if !cfg.AcceptanceConfigured() {
		return RecheckReport{}, errors.New("recheck: commands.acceptance is not configured in .kogen/project.yaml of " + in.Pinned)
	}
	rules := controlsRules{acceptance: intent.Acceptance, protect: cfg.Protect}
	if cfg.Acceptance != nil {
		rules.steps = cfg.Acceptance.StepsDir
	}
	rawControls, err := os.ReadFile(filepath.Join(in.Project, project.IntentDir(slug), "controls.json"))
	if err != nil {
		return recheckUnavailable(recheckGateControls, fmt.Sprintf("%s/controls.json cannot be read, so the correct implementation is unknown.", project.IntentDir(slug))), nil
	}
	lines := strings.Split(string(feature), "\n")
	controls, problems := parseControls(rawControls, scenarios, lines, rules)
	if controls == nil || controls.deferred != "" {
		evidence := problems
		if controls != nil {
			evidence = []string{fmt.Sprintf("%s/controls.json defers the correct implementation to the Build.", project.IntentDir(slug))}
		}
		return recheckUnavailable(recheckGateControls, evidence...), nil
	}
	draft, err := recheckDraftOverlay(in.Project, slug, intent)
	if err != nil {
		return RecheckReport{}, err
	}

	kept := map[string]bool{}
	for _, p := range in.Properties {
		if p.Kind == recheckKept {
			for _, name := range p.Scenarios {
				kept[normalizeWhitespace(name)] = true
			}
		}
	}
	isKept := func(sc project.Scenario) bool {
		return kept[normalizeWhitespace(sc.Name)] || (sc.Outline != nil && kept[normalizeWhitespace(sc.Outline.Name)])
	}

	timeout := in.Timeout
	if timeout <= 0 {
		timeout = verify.StepTimeout
	}
	s := sys.NewWith(os.Environ(), in.Pinned)
	x := controlsExec{s: s, cfg: cfg, slug: slug, clock: time.Now, acceptBound: timeout,
		materialize: func(tree string) error { return recheckCopyTree(in.Pinned, tree) }}
	command := normalizeWhitespace(cfg.Acceptance.Run)

	// runOne runs the acceptance once; a runner failure is returned as the finding.
	runOne := func(where string, overlays ...controlsOverlay) (controlsRunResult, *RecheckReport, error) {
		res, err := x.run(ctx, scenarios, false, time.Time{}, overlays...)
		var setup *controlsSetupError
		switch {
		case errors.As(err, &setup):
			tail := verify.Tail(setup.output, 20)
			evidence := []string{fmt.Sprintf("setup command `%s` exited with exit code %d %s.", normalizeWhitespace(setup.command), setup.status, where)}
			evidence = append(evidence, strings.Split(tail, "\n")...)
			return res, &RecheckReport{Verdict: RecheckRunnerFailure, RunnerKind: RunnerSetupError, Class: setupFailureClass,
				Command: setup.command, ExitCode: setup.status, Evidence: evidence,
				Text: fmt.Sprintf("Runner failure: the setup command `%s` failed with exit code %d %s; the draft was not run.", normalizeWhitespace(setup.command), setup.status, where)}, nil
		case err != nil:
			return res, nil, err
		case res.cutOff:
			return res, &RecheckReport{Verdict: RecheckRunnerFailure, RunnerKind: RunnerTimeout, Command: command, ExitCode: -1,
				Evidence: []string{fmt.Sprintf("the acceptance command `%s` did not finish within %d s %s and was stopped.", command, int(timeout.Seconds()), where)},
				Text:     fmt.Sprintf("Runner failure: the acceptance command `%s` timed out after %d s %s; no draft fault is inferred.", command, int(timeout.Seconds()), where)}, nil
		case res.accept == "":
			return res, nil, errors.New("recheck: the acceptance command did not run in the scratch copy")
		}
		fail := func(kind, why string) (controlsRunResult, *RecheckReport, error) {
			evidence := []string{fmt.Sprintf("the acceptance command `%s` %s %s.", command, why, where)}
			if out := strings.TrimSpace(verify.Tail(res.acceptOutput, 20)); out != "" {
				evidence = append(evidence, strings.Split(out, "\n")...)
			}
			return res, &RecheckReport{Verdict: RecheckRunnerFailure, RunnerKind: kind, Command: command, ExitCode: res.acceptStatus, Evidence: evidence,
				Text: fmt.Sprintf("Runner failure (%s): the acceptance command `%s` %s %s; no draft fault is inferred.", kind, command, why, where)}, nil
		}
		switch {
		case res.acceptStatus == 127:
			return fail(RunnerMissingTool, "exited with exit code 127, a tool is missing")
		case res.acceptStatus > 128 && res.acceptStatus < 160:
			return fail(RunnerKilledCommand, fmt.Sprintf("was killed by %s (exit code %d)", syscall.Signal(res.acceptStatus-128), res.acceptStatus))
		case !res.complete:
			return fail(RunnerIncompleteReport, fmt.Sprintf("exited with exit code %d without a readable report of every scenario", res.acceptStatus))
		}
		return res, nil, nil
	}

	base, rep, err := runOne("on the unchanged tree", draft)
	if err != nil || rep != nil {
		return reportOr(rep), err
	}
	baseFail := map[int]recheckFailure{}
	for k, sc := range scenarios {
		f, failed := recheckFailureOf(base, sc, lines)
		if !failed {
			continue
		}
		if isKept(sc) {
			evidence := []string{fmt.Sprintf("on the unchanged tree: %s", f.line())}
			return RecheckReport{Verdict: RecheckBadDraft, Scenario: f.scenario, Command: command, ExitCode: base.acceptStatus, Evidence: evidence,
				Text: fmt.Sprintf("Bad draft: the kept scenario %q fails on the unchanged tree, so it does not assert the unchanged behaviour.", f.scenario)}, nil
		}
		baseFail[k] = f
	}

	ref, rep, err := runOne("with the reference in place", draft, controls.correct)
	if err != nil || rep != nil {
		return reportOr(rep), err
	}
	same := -1
	for k, sc := range scenarios {
		if isKept(sc) {
			continue
		}
		f, failed := recheckFailureOf(ref, sc, lines)
		if !failed {
			continue
		}
		if b, onBase := baseFail[k]; onBase && b.step == f.step && b.message == f.message {
			if same < 0 {
				same = k
			}
			continue
		}
		text := referenceFailureLines([]controlsProp{{scenarios: []int{k}}}, scenarios, lines, ref)
		evidence := []string{"with the reference in place: " + f.line()}
		if b, onBase := baseFail[k]; onBase {
			evidence = append(evidence, "on the unchanged tree: "+b.line())
		} else {
			evidence = append(evidence, fmt.Sprintf("on the unchanged tree: scenario %q passes.", f.scenario))
		}
		return RecheckReport{Verdict: RecheckReferenceFail, Scenario: f.scenario, Command: command, ExitCode: ref.acceptStatus, Evidence: evidence,
			Text: strings.Join(text, "\n")}, nil
	}
	if same >= 0 {
		f := baseFail[same]
		return RecheckReport{AttributionOpen: true, Scenario: f.scenario, Command: command, ExitCode: ref.acceptStatus,
			Evidence: []string{"on the unchanged tree: " + f.line(), "with the reference in place: " + f.line()},
			Text:     fmt.Sprintf("Attribution is open: scenario %q fails the same way on the unchanged tree and with the reference in place, and is not a kept scenario.", f.scenario)}, nil
	}
	return RecheckReport{Text: "No fault found: every scenario passes with the reference in place, or fails only on the unchanged tree."}, nil
}

func reportOr(rep *RecheckReport) RecheckReport {
	if rep == nil {
		return RecheckReport{}
	}
	return *rep
}

// recheckFailure is how one scenario failed: the step it failed at and its failure message, with
// the scratch copy's path left out so that two runs can be compared.
type recheckFailure struct {
	scenario, step, message string
}

func (f recheckFailure) line() string {
	return fmt.Sprintf("scenario %q failed at %q: %s", f.scenario, f.step, f.message)
}

func recheckFailureOf(run controlsRunResult, sc project.Scenario, feature []string) (recheckFailure, bool) {
	c := run.cases[sc.Line]
	if c.status() != "failed" {
		return recheckFailure{}, false
	}
	first := c.firstUnpassed()
	message := first.message
	for _, root := range run.roots {
		message = strings.ReplaceAll(message, root, "<tree>")
	}
	return recheckFailure{scenario: normalizeWhitespace(sc.Name), step: controlsStepLabel(first, feature), message: controlsFailureMessage(message)}, true
}

// recheckDraftOverlay is the draft of slug in root as an overlay: every file of its Intent
// directory and the acceptance files its Intent lists.
func recheckDraftOverlay(root, slug string, intent *project.Intent) (controlsOverlay, error) {
	o := controlsOverlay{files: map[string]string{}}
	add := func(rel string) error {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
		if errors.Is(err, fs.ErrNotExist) || (err == nil && !info.Mode().IsRegular()) {
			return nil
		}
		if err != nil {
			return err
		}
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		o.files[rel] = string(b)
		return err
	}
	dir := project.IntentDir(slug)
	err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(dir)), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		return add(filepath.ToSlash(rel))
	})
	if err != nil {
		return o, err
	}
	listed := intent.StepFiles()
	sort.Strings(listed)
	for _, rel := range listed {
		if err := add(rel); err != nil {
			return o, err
		}
	}
	return o, nil
}

// recheckCopyTree copies the checkout src to the new directory dst without .git and the local
// state of Kogen, keeping symlinks as links.
func recheckCopyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if slash == ".git" || slash == ".kogen/local" {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return os.MkdirAll(target, info.Mode().Perm()|0o700)
		case info.Mode()&fs.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case info.Mode().IsRegular():
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			return os.WriteFile(target, b, info.Mode().Perm())
		}
		return nil
	})
}
