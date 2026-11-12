package shape

import (
	"slices"
	"strings"
	"testing"
)

const verbatimStatement = "`kogen uninstall` keeps logins and `.kogen/host-id`."

const verbatimRequest = "Docs task.\n" + verbatimStatement + "\nKeep the page short."

func verbatimIntent(line string) string { return "# Docs\n\n## Decisions\n" + line + "\n" }

func TestVerbatimStatementIsNeverRefused(t *testing.T) {
	parallel(t)
	intent := verbatimIntent("- " + verbatimStatement + " (source: request)")
	item := decisionItem{statement: verbatimStatement, quotes: []string{verbatimStatement}, verdict: "unsupported", why: "Stated in other facts."}
	event, lines := decisionOutcome([]decisionItem{item}, verbatimRequest, intent)
	if lines != nil {
		t.Fatalf("a statement the request says literally was refused: %q", lines)
	}
	got := event["items"].([]any)[0].(map[string]any)
	if got["model_verdict"] != "unsupported" || got["advisory"] != true || event["unsupported"] != 0 {
		t.Fatalf("event %v", event)
	}
	// Without request_quotes the literal text still carries it.
	item.quotes = nil
	if _, lines := decisionOutcome([]decisionItem{item}, verbatimRequest, intent); lines != nil {
		t.Fatalf("refused without quotes: %q", lines)
	}
}

func TestParaphrasedUnsupportedStatementIsStillRefused(t *testing.T) {
	parallel(t)
	const statement = "Uninstall leaves logins and the host id alone."
	intent := verbatimIntent("- " + statement + " (source: request)")
	item := decisionItem{statement: statement, verdict: "unsupported", why: "w"}
	_, lines := decisionOutcome([]decisionItem{item}, verbatimRequest, intent)
	if lines != nil {
		// With neither evidence nor assumption Kogen marks it; the plain line must not be refused for being verbatim.
		t.Logf("feedback %q", lines)
	}
	event, _ := decisionOutcome([]decisionItem{item}, verbatimRequest, intent)
	got := event["items"].([]any)[0].(map[string]any)
	if got["verdict"] != "unsupported" || got["advisory"] == true || event["unsupported"] != 1 {
		t.Fatalf("paraphrase was supported: %v", event)
	}
	// Quoted request text that does not state it keeps the refusal.
	item.quotes = []string{"Keep the page short."}
	_, lines = decisionOutcome([]decisionItem{item}, verbatimRequest, intent)
	if len(lines) == 0 || !strings.Contains(strings.Join(lines, "\n"), "verdict: unsupported") {
		t.Fatalf("paraphrase with quote not refused: %q", lines)
	}
}

func TestVerbatimStatementContradictingQuotedRequestIsRefused(t *testing.T) {
	parallel(t)
	const request = "Uninstall keeps logins.\nUninstall removes the host id."
	const statement = "Uninstall keeps logins."
	intent := verbatimIntent("- " + statement + " (source: request)")
	item := decisionItem{statement: statement, quotes: []string{"Uninstall removes the host id."}, verdict: "contradicts",
		intentValue: "keeps logins", requestValue: "removes the host id", difference: "d", why: "w"}
	event, lines := decisionOutcome([]decisionItem{item}, request, intent)
	if len(lines) == 0 || event["contradicts"] != 1 || !strings.Contains(strings.Join(lines, "\n"), "verdict: contradicts") {
		t.Fatalf("contradiction escaped: %q %v", lines, event)
	}
}

func TestAssumptionFormatLineOnlyForMalformedMarker(t *testing.T) {
	parallel(t)
	const request = "Docs task."
	const plain = "Uninstall keeps the host id."
	// Ungrounded statement the request is silent on, evidence-free and not markable (no source tag): refused without the rule.
	item := decisionItem{statement: plain, quotes: []string{"Docs task."}, verdict: "unsupported", why: "w"}
	_, lines := decisionOutcome([]decisionItem{item}, request, verbatimIntent("- "+plain+" (source: request)"))
	if len(lines) == 0 || slices.Contains(lines, assumptionRule) {
		t.Fatalf("rule shown without a marker: %q", lines)
	}
	// A malformed marker: wrong bracket and wrong place.
	bad := "- " + plain + " (source: request) [assumption: confirm]"
	item = decisionItem{statement: bad, quotes: []string{"Docs task."}, verdict: "unsupported", why: "w"}
	_, lines = decisionOutcome([]decisionItem{item}, request, verbatimIntent(bad))
	if len(lines) == 0 || !slices.Contains(lines, assumptionRule) {
		t.Fatalf("rule missing for malformed marker: %q", lines)
	}
}
