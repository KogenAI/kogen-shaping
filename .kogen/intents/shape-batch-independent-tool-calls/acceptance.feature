Feature: Batch independent shaping tool calls
  This feature drives the real native Shape entry point with the existing scripted
  provider in a disposable copy of the candidate. Slow reads are FIFOs under the
  ignored .kogen/local folder whose writers answer only once the expected number of
  readers is open at the same time (or after two seconds), so overlapping reads are
  observed without wall-clock thresholds.

  Scenario: Read-only calls of one response overlap and answer in call order
    One response calls read_file .kogen/local/acceptance-batch/a.txt, search_text
    "tiny" in README.md, read_file b.txt and read_file c.txt. All three reads must be
    in flight together. The writers answer c, then b, then a, yet the results reach
    the next request in call order read-a, search, read-b, read-c with the texts
    "1<tab>alpha", the README match "A tiny calculator.", "1<tab>beta" and "1<tab>gamma".
    Then shape-batch-independent-tool-calls: the overlap case passes

  Scenario: Probes stay in order around a concurrent read group
    One response calls probe p1, read_file x.txt, read_file y.txt and probe p2. Each
    probe logs its start, sleeps 0.4 s and logs its end. Both reads must start after
    p1 ended and be in flight together, p2 must start after both reads were answered,
    the probes never overlap, and the results arrive in call order.
    Then shape-batch-independent-tool-calls: the barriers case passes

  Scenario: An unchanged repeat read in the same window answers a short note
    The shaper proposes .kogen/intents/is-even/intent.md, reads its custody file
    .kogen/local/shaping/is-even/proposed/.kogen/intents/is-even/intent.md and
    README.md, then reads both again plus README.md lines 3 to 3. The repeats answer
    the note for the earlier result; the first read of the line range answers
    "3<tab>A tiny calculator.". After a second proposal changes the custody file its
    next read returns the new numbered content, while README.md still answers the note.
    Then shape-batch-independent-tool-calls: a repeated unchanged read of README.md answers "already read in this window, unchanged (32 bytes, sha 28ca9705)"

  Scenario: The shaper is told to batch independent calls
    Then shape-batch-independent-tool-calls: shaper.md, the shaper instructions and the read_file, search_text, list_files and probe descriptions contain "Request every independent read_file, search_text, list_files and probe call you need in one response, and do not re-read a file you have already read unless it changed."
