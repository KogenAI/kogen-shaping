package shape

import "slices"

// gateProblems are the problems one gate refused a submission with.
type gateProblems struct {
	gate     string
	problems []string
}

type retryBudget struct {
	limit, consecutive, total int
	gate                      string
	// last is the refusal the previous submission got: every gate with its problems.
	last []gateProblems
}

// refuse counts a submission refused by one gate: the same gate as before is no progress.
func (b *retryBudget) refuse(gate string) bool {
	return b.refuseSet([]gateProblems{{gate: gate}})
}

// refuseSet counts a submission refused by every gate in set, the first gate being the one that
// names the refusal. A refusal by one gate repeats the previous one when it is the same gate. When
// either refusal came from several gates, it repeats the previous one only if every problem of the
// previous one is still there: fixing some problems is progress, whichever gate comes first.
func (b *retryBudget) refuseSet(set []gateProblems) bool {
	repeat := b.total > 0 && len(set) > 0 && set[0].gate == b.gate
	if b.total > 0 && (len(set) > 1 || len(b.last) > 1) {
		repeat = stillRefused(b.last, set)
	}
	if repeat {
		b.consecutive++
	} else {
		b.consecutive = 1
	}
	if len(set) > 0 {
		b.gate = set[0].gate
	}
	b.last = set
	b.total++
	return b.consecutive >= b.limit || b.total >= 2*b.limit
}

// stillRefused is whether every gate of previous refuses again with at least all its problems.
func stillRefused(previous, now []gateProblems) bool {
	for _, p := range previous {
		i := slices.IndexFunc(now, func(n gateProblems) bool { return n.gate == p.gate })
		if i < 0 {
			return false
		}
		for _, problem := range p.problems {
			if !slices.Contains(now[i].problems, problem) {
				return false
			}
		}
	}
	return true
}

// referenceUnrepairedLine ends the refusal of a draft that still fails with the reference in place
// after the extra round its first reference failure was granted.
const referenceUnrepairedLine = "reference and test were not repaired in the extra round"

// refuseReference counts a submission refused because the correct implementation of controls.json
// fails a scenario. The first such failure is never a repeated failure and never ends the shaping,
// whatever was refused before: the shaper gets an extra round. A later one (again true) is counted
// as any refusal and ends the shaping.
func (b *retryBudget) refuseReference(set []gateProblems, again bool) bool {
	if again {
		b.refuseSet(set)
		return true
	}
	b.consecutive = 1
	if len(set) > 0 {
		b.gate = set[0].gate
	}
	b.last = set
	b.total++
	return false
}
