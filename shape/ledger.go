package shape

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// ---- the decision ledger: the sourced decisions of a shaping, kept across its rounds ----

// ledgerFile is `.kogen/local/shaping/<slug>/decisions.json`, beside state.json.
const ledgerFile = "decisions.json"

// Receipt kinds: a file this shaping's main conversation read, a file the external session's
// inspection covered (its text is read from the checkout), and the output of a probe.
const (
	receiptRead     = "read"
	receiptCheckout = "checkout"
	receiptProbe    = "probe"
)

// receipt is the evidence one cited item had when a decision was recorded.
type receipt struct {
	Item   string `json:"item"`
	Kind   string `json:"kind"`
	Round  int    `json:"round"`
	SHA256 string `json:"sha256"`
	Text   string `json:"text"`
}

// recordedDecision is one decision of the ledger; it stands while SupersededBy is empty.
type recordedDecision struct {
	ID           string    `json:"id"`
	Round        int       `json:"round"`
	Decision     string    `json:"decision"`
	Constraints  []string  `json:"constraints"`
	Uncertainty  string    `json:"uncertainty"`
	Revises      string    `json:"revises"`
	Why          string    `json:"why"`
	SupersededBy string    `json:"superseded_by"`
	Evidence     []receipt `json:"evidence"`
}

func (d recordedDecision) stands() bool { return d.SupersededBy == "" }

// decisionLedger is the decisions of a shaping in the order they were recorded. A nil ledger
// (Refine's) holds none.
type decisionLedger struct {
	Decisions []recordedDecision
}

type ledgerDoc struct {
	Version   int                `json:"version"`
	Decisions []recordedDecision `json:"decisions"`
}

// readLedger is the ledger kept beside state.json, empty when there is none.
func readLedger(dir string) (*decisionLedger, error) {
	path := filepath.Join(dir, ledgerFile)
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &decisionLedger{}, nil
	}
	if err != nil {
		return nil, err
	}
	var doc ledgerDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &decisionLedger{Decisions: doc.Decisions}, nil
}

// save writes the ledger whole, privately.
func (l *decisionLedger) save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	doc := ledgerDoc{Version: 1, Decisions: slices.Clone(l.Decisions)}
	for i := range doc.Decisions {
		d := &doc.Decisions[i]
		if d.Constraints == nil {
			d.Constraints = []string{}
		}
		if d.Evidence == nil {
			d.Evidence = []receipt{}
		}
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(doc); err != nil {
		return err
	}
	path := filepath.Join(dir, ledgerFile)
	if err := os.WriteFile(path, b.Bytes(), 0o666); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func (l *decisionLedger) empty() bool { return l == nil || len(l.Decisions) == 0 }

// standing are the decisions that stand, in ledger order.
func (l *decisionLedger) standing() []recordedDecision {
	if l == nil {
		return nil
	}
	var out []recordedDecision
	for _, d := range l.Decisions {
		if d.stands() {
			out = append(out, d)
		}
	}
	return out
}

// standingByID is the decision that stands with the id.
func (l *decisionLedger) standingByID(id string) (recordedDecision, bool) {
	for _, d := range l.standing() {
		if d.ID == id {
			return d, true
		}
	}
	return recordedDecision{}, false
}

// fileReceipts are the receipts of a file, by cleaned path, in ledger and item order.
func (l *decisionLedger) fileReceipts(cleaned string) []receipt {
	if l == nil {
		return nil
	}
	var out []receipt
	for _, d := range l.Decisions {
		for _, r := range d.Evidence {
			if r.Kind == receiptProbe {
				continue
			}
			if it := parseEvidenceItem(r.Item); it.command == "" && !it.empty && cleanCitedPath(it.path) == cleaned {
				out = append(out, r)
			}
		}
	}
	return out
}

// fileReceipt is a receipt of the file whose digest is the file's current one (the last such),
// or, when the ledger holds only receipts with another digest, the last of them with changed true.
func (l *decisionLedger) fileReceipt(cleaned, digest string) (r receipt, found, changed bool) {
	all := l.fileReceipts(cleaned)
	for i := len(all) - 1; i >= 0; i-- {
		if digest != "" && all[i].SHA256 == digest {
			return all[i], true, false
		}
	}
	if len(all) > 0 {
		return all[len(all)-1], false, true
	}
	return receipt{}, false, false
}

// probeReceipt is the last receipt of the probe item, as cited (trimmed).
func (l *decisionLedger) probeReceipt(item string) (receipt, bool) {
	if l == nil {
		return receipt{}, false
	}
	var out receipt
	found := false
	for _, d := range l.Decisions {
		for _, r := range d.Evidence {
			if r.Kind == receiptProbe && r.Item == item {
				out, found = r, true
			}
		}
	}
	return out, found
}

// probeReceiptOf is the last probe receipt of the command.
func (l *decisionLedger) probeReceiptOf(command string) (receipt, bool) {
	if l == nil {
		return receipt{}, false
	}
	var out receipt
	found := false
	for _, d := range l.Decisions {
		for _, r := range d.Evidence {
			if r.Kind == receiptProbe && parseEvidenceItem(r.Item).command == command {
				out, found = r, true
			}
		}
	}
	return out, found
}

// ---- rendering ----

// intentLine is the Decisions line that keeps the decision in the Intent.
func (d recordedDecision) intentLine() string {
	var b strings.Builder
	fmt.Fprintf(&b, "- %s (decision: %s, round %d)", d.Decision, d.ID, d.Round)
	for _, c := range d.Constraints {
		fmt.Fprintf(&b, " (constraint: %s)", c)
	}
	if len(d.Evidence) > 0 {
		items := make([]string, len(d.Evidence))
		for i, r := range d.Evidence {
			items[i] = r.Item
		}
		fmt.Fprintf(&b, " (evidence: %s)", strings.Join(items, "; "))
	}
	if strings.TrimSpace(d.Uncertainty) != "" {
		fmt.Fprintf(&b, " (assumption: %s)", d.Uncertainty)
	}
	if d.Revises != "" {
		fmt.Fprintf(&b, " (revises: %s because %s)", d.Revises, d.Why)
	}
	b.WriteString(" (source: request)")
	return b.String()
}

// digestOf is the SHA-256 of a file's current checkout bytes by cleaned path, "" when it
// cannot be read.
type digestOf func(cleaned string) string

// paren says where a receipt's text came from and whether the file has changed since.
func (r receipt) paren(current digestOf) string {
	switch r.Kind {
	case receiptProbe:
		return fmt.Sprintf("(probe output in round %d)", r.Round)
	case receiptCheckout:
		return fmt.Sprintf("(read from the checkout in round %d%s)", r.Round, r.changedNote(current))
	}
	return fmt.Sprintf("(read by this shaping in round %d%s)", r.Round, r.changedNote(current))
}

func (r receipt) changedNote(current digestOf) string {
	if current(cleanCitedPath(parseEvidenceItem(r.Item).path)) != r.SHA256 {
		return "; changed since"
	}
	return ""
}

// summary is the decision with its constraints, uncertainty, revision and evidence, headed by head.
func (d recordedDecision) summary(head string, current digestOf) string {
	lines := []string{head + ": " + d.Decision}
	for _, c := range d.Constraints {
		lines = append(lines, "   constraint: "+c)
	}
	if strings.TrimSpace(d.Uncertainty) != "" {
		lines = append(lines, "   uncertainty: "+d.Uncertainty)
	}
	if d.Revises != "" {
		lines = append(lines, fmt.Sprintf("   revises: %s because %s", d.Revises, d.Why))
	}
	for _, r := range d.Evidence {
		lines = append(lines, "   evidence: "+r.Item+" "+r.paren(current))
	}
	return strings.Join(lines, "\n")
}

// checkInputText is the question check's "# Recorded decisions" section, "" without a decision
// that stands.
func (l *decisionLedger) checkInputText(current digestOf) string {
	var blocks []string
	for _, d := range l.standing() {
		blocks = append(blocks, d.summary(fmt.Sprintf("%s (round %d)", d.ID, d.Round), current))
		for _, r := range d.Evidence {
			blocks = append(blocks, fmt.Sprintf("%s cites %s %s:\n%s", d.ID, r.Item, r.paren(current), strings.TrimRight(r.Text, "\n")))
		}
	}
	if len(blocks) == 0 {
		return ""
	}
	return "\n\n# Recorded decisions\n\n" + strings.Join(blocks, "\n\n")
}

const contextDecisionsHeading = "\n# Your recorded decisions\n\nKogen writes the standing lines into the Intent's Decisions. Follow their constraints and leave ledger identities to Kogen. Each decision stands until a revision cites evidence this round read or ran.\n\n"

// contextText is the shaper's context section of the recorded decisions, "" for an empty ledger.
func (l *decisionLedger) contextText(current digestOf) string {
	if l.empty() {
		return ""
	}
	blocks := make([]string, len(l.Decisions))
	for i, d := range l.Decisions {
		if d.stands() {
			blocks[i] = d.summary(fmt.Sprintf("%s (round %d)", d.ID, d.Round), current) + "\n   Intent line: " + d.intentLine()
		} else {
			blocks[i] = d.summary(fmt.Sprintf("%s (round %d, revised by %s)", d.ID, d.Round, d.SupersededBy), current)
		}
	}
	return contextDecisionsHeading + strings.Join(blocks, "\n\n") + "\n"
}

// ---- the submitted decisions ----

// sourcedDecision is one decision of a submission's decisions list.
type sourcedDecision struct {
	decision    string
	evidence    []string
	constraints []string
	uncertainty string
	revises     string
	why         string
}

const (
	decisionsWithBlocked = "decisions go with questions or files, not with a blocked report"
	decisionsMalformed   = "decisions: every decision needs decision text, optional evidence and constraints lists of text, and optional uncertainty, revises and why text"
)

var recordedDecisionID = regexp.MustCompile(`^D[1-9][0-9]*$`)

// decisionsOf reads the submission's decisions: none for an absent or empty list, otherwise the
// decisions, or ok false when the value is malformed.
func decisionsOf(args map[string]any) (list []sourcedDecision, present, ok bool) {
	raw, has := args["decisions"]
	if !has {
		return nil, false, true
	}
	items, isList := raw.([]any)
	if isList && len(items) == 0 {
		return nil, false, true
	}
	if !isList {
		return nil, true, false
	}
	for _, it := range items {
		m, isMap := it.(map[string]any)
		if !isMap {
			return nil, true, false
		}
		text, isStr := m["decision"].(string)
		if !isStr || strings.TrimSpace(text) == "" {
			return nil, true, false
		}
		d := sourcedDecision{decision: text}
		var okE, okC, okU, okR, okW bool
		if d.evidence, okE = textList(m, "evidence"); !okE {
			return nil, true, false
		}
		if d.constraints, okC = textList(m, "constraints"); !okC {
			return nil, true, false
		}
		if d.uncertainty, okU = optionalText(m, "uncertainty"); !okU {
			return nil, true, false
		}
		if d.revises, okR = optionalText(m, "revises"); !okR {
			return nil, true, false
		}
		if d.why, okW = optionalText(m, "why"); !okW {
			return nil, true, false
		}
		d.revises = strings.TrimSpace(d.revises)
		if strings.TrimSpace(d.why) == "" {
			id, why, inline := strings.Cut(d.revises, " because ")
			if inline && recordedDecisionID.MatchString(id) && strings.TrimSpace(why) != "" {
				d.revises, d.why = id, strings.TrimSpace(why)
			}
		}
		list = append(list, d)
	}
	return list, true, true
}

// textList is the list of strings under key: nil when absent, ok false when it is not one.
func textList(m map[string]any, key string) ([]string, bool) {
	raw, has := m[key]
	if !has {
		return nil, true
	}
	items, isList := raw.([]any)
	if !isList {
		return nil, false
	}
	out := make([]string, len(items))
	for i, it := range items {
		s, isStr := it.(string)
		if !isStr {
			return nil, false
		}
		out[i] = s
	}
	return out, true
}

func optionalText(m map[string]any, key string) (string, bool) {
	raw, has := m[key]
	if !has {
		return "", true
	}
	s, ok := raw.(string)
	return s, ok
}

// sameAs is whether the submitted decision equals the recorded one in every field.
func (d sourcedDecision) sameAs(r recordedDecision) bool {
	if d.decision != r.Decision || d.uncertainty != r.Uncertainty || d.revises != r.Revises || d.why != r.Why ||
		!slices.Equal(d.constraints, r.Constraints) || len(d.evidence) != len(r.Evidence) {
		return false
	}
	for i, item := range d.evidence {
		if strings.TrimSpace(item) != r.Evidence[i].Item {
			return false
		}
	}
	return true
}

// ---- the checkout facts the ledger is judged against ----

// checkoutBytes are the bytes of a tracked file in the checkout, or at HEAD when the checkout
// does not hold it.
func (sh *shaping) checkoutBytes(cleaned string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(sh.root, filepath.FromSlash(cleaned)))
	if err == nil {
		return data, nil
	}
	out, gerr := sh.repo.OK("show", "HEAD:"+cleaned)
	if gerr != nil {
		return nil, gerr
	}
	return []byte(out), nil
}

// checkoutDigest is the SHA-256 of the file's current checkout bytes, "" when unreadable.
func (sh *shaping) checkoutDigest(cleaned string) string {
	data, err := sh.checkoutBytes(cleaned)
	if err != nil {
		return ""
	}
	return ContentHash(data)
}

// headFiles lists the tracked file paths once for this shaping.
func (sh *shaping) headFiles() (map[string]bool, error) {
	if sh.tracked == nil {
		out, err := sh.repo.OK("ls-tree", "-r", "-z", "--name-only", "HEAD")
		if err != nil {
			return nil, err
		}
		sh.tracked = map[string]bool{}
		for _, f := range strings.Split(strings.TrimSuffix(out, "\x00"), "\x00") {
			sh.tracked[f] = true
		}
	}
	return sh.tracked, nil
}

// trackedAtHead is whether the cleaned path is a file tracked at HEAD.
func (sh *shaping) trackedAtHead(cleaned string) (bool, error) {
	tracked, err := sh.headFiles()
	return tracked[cleaned], err
}

// ---- the decision record gate ----

const decisionRecordHeadline = "Sourced decisions: %d problem(s) with the decisions this submission records; nothing was recorded, asked or published. " +
	"A sourced decision cites, as evidence items, files this shaping read (or that an earlier round read and that are unchanged since) or ran `<command>` for a probe of this shaping that succeeded; where there is no such evidence, state the uncertainty you decided under instead. " +
	"A revision names the decision it revises in revises, says why it changed and cites evidence this round read or ran."

const probeSucceeded = "exit status: 0\ntimed_out: false\n"

// decisionReceipt checks one evidence item of decision n: the receipt it earns, whether it is
// fresh (made from this round's read or probe), or the one problem line.
func (sh *shaping) decisionReceipt(n int, raw string) (r receipt, fresh bool, problem string, err error) {
	item := strings.TrimSpace(raw)
	it := parseEvidenceItem(raw)
	at := fmt.Sprintf("- sourced decision %d: ", n)
	switch {
	case it.empty:
		return r, false, at + "an empty evidence item", nil
	case it.command != "":
		if sh.ext != nil {
			return r, false, at + fmt.Sprintf("Kogen keeps no output of the external session's commands, so `%s` is no evidence; cite a file this session read", it.command), nil
		}
		if sh.probed[it.command] {
			text := sh.probeTexts[it.command]
			if !strings.HasPrefix(text, probeSucceeded) {
				first, _, _ := strings.Cut(text, "\n")
				return r, false, at + fmt.Sprintf("the probe of `%s` failed (%s), so it is no evidence", it.command, first), nil
			}
			return receipt{Item: item, Kind: receiptProbe, Round: sh.round, Text: text}, true, "", nil
		}
		if c, ok := sh.ledger.probeReceipt(item); ok {
			return c, false, "", nil
		}
		return r, false, at + fmt.Sprintf("no probe of this shaping ran `%s`", it.command), nil
	}
	cleaned := cleanCitedPath(it.path)
	if strings.HasSuffix(it.path, "/") {
		return r, false, at + it.path + " is a directory; cite the files this shaping read", nil
	}
	tracked, err := sh.headFiles()
	if err != nil {
		return r, false, "", err
	}
	if !tracked[cleaned] {
		if headDirectory(tracked, cleaned) {
			return r, false, at + it.path + " is a directory; cite the files this shaping read", nil
		}
		return r, false, at + it.path + " is not a file tracked at HEAD", nil
	}
	data, err := sh.checkoutBytes(cleaned)
	if err != nil {
		return r, false, "", err
	}
	digest := ContentHash(data)
	carried, found, changed := sh.ledger.fileReceipt(cleaned, digest)
	// A carried receipt keeps the earlier kind, round, digest and text under the item as cited now.
	carried.Item = item
	changedLine := at + fmt.Sprintf("%s changed since round %d read it; read it again", it.path, carried.Round)
	if sh.ext != nil {
		switch {
		case found && !sh.externalReads[cleaned]:
			return carried, false, "", nil
		case sh.inspected:
			return receipt{Item: item, Kind: receiptCheckout, Round: sh.round, SHA256: digest, Text: numberedLines(string(data))}, true, "", nil
		case changed:
			return r, false, changedLine, nil
		}
		return r, false, at + fmt.Sprintf("this shaping's session read nothing, so %s was not read", it.path), nil
	}
	if text, ok := sh.readTexts[cleaned]; ok && !strings.HasPrefix(text, "error: ") {
		return receipt{Item: item, Kind: receiptRead, Round: sh.round, SHA256: digest, Text: text}, true, "", nil
	}
	switch {
	case found:
		return carried, false, "", nil
	case changed:
		return r, false, changedLine, nil
	}
	return r, false, at + "this shaping did not read " + it.path, nil
}

// decisionRecordGate checks and records the decisions of a counted questions or files
// submission. ok is true for a refusal.
func (sh *shaping) decisionRecordGate(submitted []sourcedDecision) (string, outcome, bool, error) {
	if len(submitted) == 0 || sh.ledger == nil {
		return "", outcome{}, false, nil
	}
	type checked struct {
		d        sourcedDecision
		receipts []receipt
	}
	var problems []string
	var added []checked
	var keptIDs []string
	used := map[string]bool{}
	revised := map[string]bool{}
	for i, d := range submitted {
		n := i + 1
		same := false
		for _, r := range sh.ledger.standing() {
			if !used[r.ID] && d.sameAs(r) {
				used[r.ID] = true
				keptIDs = append(keptIDs, r.ID)
				same = true
				break
			}
		}
		if same {
			continue
		}
		at := fmt.Sprintf("- sourced decision %d: ", n)
		if len(d.evidence) == 0 && strings.TrimSpace(d.uncertainty) == "" {
			problems = append(problems, at+"no evidence and no uncertainty; cite what this shaping read or ran, or state the uncertainty you decided under")
		}
		revisesStands := false
		if d.revises != "" {
			if strings.TrimSpace(d.why) == "" {
				problems = append(problems, at+"revises "+d.revises+" without why")
			}
			if _, ok := sh.ledger.standingByID(d.revises); ok && !revised[d.revises] {
				revisesStands = true
			} else {
				problems = append(problems, at+d.revises+" is not a recorded decision that stands")
			}
		}
		c := checked{d: d}
		anyFresh := false
		for _, raw := range d.evidence {
			r, fresh, problem, err := sh.decisionReceipt(n, raw)
			if err != nil {
				return "", outcome{}, false, err
			}
			if problem != "" {
				problems = append(problems, problem)
				continue
			}
			anyFresh = anyFresh || fresh
			c.receipts = append(c.receipts, r)
		}
		if revisesStands {
			revised[d.revises] = true
			if !anyFresh {
				problems = append(problems, at+fmt.Sprintf("revises %s without evidence this round read or ran, so %s stands", d.revises, d.revises))
			}
		}
		added = append(added, c)
	}
	if len(problems) > 0 {
		sh.refused("decision_record", problems)
		text := strings.Join(append([]string{fmt.Sprintf(decisionRecordHeadline, len(problems))}, problems...), "\n")
		return text, outcome{kind: outInvalid, problems: problems}, true, nil
	}
	recorded, kept := []string{}, []string{}
	for _, c := range added {
		id := fmt.Sprintf("D%d", len(sh.ledger.Decisions)+1)
		d := recordedDecision{
			ID: id, Round: sh.round, Decision: c.d.decision, Constraints: c.d.constraints, Uncertainty: c.d.uncertainty,
			Revises: c.d.revises, Why: c.d.why, Evidence: c.receipts,
		}
		if d.Constraints == nil {
			d.Constraints = []string{}
		}
		if d.Evidence == nil {
			d.Evidence = []receipt{}
		}
		if d.Revises != "" {
			for k := range sh.ledger.Decisions {
				if sh.ledger.Decisions[k].ID == d.Revises {
					sh.ledger.Decisions[k].SupersededBy = id
				}
			}
		}
		sh.ledger.Decisions = append(sh.ledger.Decisions, d)
		recorded = append(recorded, id)
	}
	kept = append(kept, keptIDs...)
	if err := sh.ledger.save(sh.stateDir); err != nil {
		return "", outcome{}, false, err
	}
	sh.ev.Record(map[string]any{"event": "decisions_recorded", "round": sh.round, "recorded": recorded, "kept": kept})
	return "", outcome{}, false, nil
}

// ---- the Intent carries the decisions that stand ----

const carriedDecisionsHeadline = "Recorded decisions: %d ledger issue(s) in the Intent's Decisions; nothing was published. " +
	"A recorded decision stands until a revision that cites evidence this round read or ran replaces it, and its line is kept exactly, before its source tag, so its evidence and constraints stay in force:"

// carriedDecisionsGate verifies ledger identities after the candidate's fixers have run.
func (sh *shaping) carriedDecisionsGate(intent string) (string, outcome, bool) {
	standing := sh.ledger.standing()
	canonical := map[string]string{}
	for _, d := range standing {
		canonical[decisionLineKey(d.intentLine())] = d.ID
	}
	have := map[string]bool{}
	var problems []string
	for _, line := range decisionLines(intent) {
		key := decisionLineKey(line)
		if id, ok := canonical[key]; ok {
			if have[key] {
				problems = append(problems, "- "+id+" appears more than once as a recorded decision")
			}
			have[key] = true
			continue
		}
		refs := decisionRoundRe.FindAllStringSubmatch(line, -1)
		switch len(refs) {
		case 0:
		case 1:
			if _, ok := sh.ledger.standingByID(refs[0][1]); !ok {
				problems = append(problems, "- "+refs[0][1]+" is not a recorded decision that stands; remove its identity or submit a sourced decision")
			} else {
				problems = append(problems, "- "+refs[0][1]+" does not match its recorded line")
			}
		default:
			problems = append(problems, "- use one recorded decision per line: "+line)
		}
	}
	for _, d := range standing {
		line := d.intentLine()
		if !have[decisionLineKey(line)] {
			problems = append(problems, fmt.Sprintf("- %s (round %d) is not in Decisions; keep it as: %s", d.ID, d.Round, line))
		}
	}
	if len(problems) == 0 {
		return "", outcome{}, false
	}
	sh.refused("carried_decisions", problems)
	text := strings.Join(append([]string{fmt.Sprintf(carriedDecisionsHeadline, len(problems))}, problems...), "\n")
	return text, outcome{kind: outInvalid, problems: problems}, true
}
