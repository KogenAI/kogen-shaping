package shape

type retryBudget struct {
	limit, consecutive, total int
	gate                      string
}

func (b *retryBudget) refuse(gate string) bool {
	if b.total > 0 && gate == b.gate {
		b.consecutive++
	} else {
		b.consecutive = 1
	}
	b.gate = gate
	b.total++
	return b.consecutive >= b.limit || b.total >= 2*b.limit
}
