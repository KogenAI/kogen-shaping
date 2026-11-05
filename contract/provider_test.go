package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// turn is one scripted answer of the provider: the items of a completed response, or a stall.
type turn struct {
	items []map[string]any
	stall bool
}

// scripted is an in-process http.RoundTripper that answers on the Responses wire: no listener, no
// network. Requests offering one of the three review tools are answered outside the script; every
// other request takes the next turn, and past the script answers HTTP 400.
type scripted struct {
	clock *clock

	mu        sync.Mutex
	script    []turn
	next      int
	count     int
	remaining []time.Duration // time left to each request's deadline
	missing   int             // requests without a deadline
	shaper    int             // requests that took the script's turns or its failure
}

func newScripted(clock *clock, script ...turn) *scripted {
	return &scripted{clock: clock, script: script}
}

// clock is the client's controllable clock.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock() *clock { return &clock{now: time.Now()} }

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

func call(id, name string, args any) map[string]any {
	b, _ := json.Marshal(args)
	return map[string]any{"type": "function_call", "call_id": id, "name": name, "arguments": string(b)}
}

func answer(items ...map[string]any) turn { return turn{items: items} }

// stalled is a provider that takes a request, says nothing and loses the connection once more than
// thirty minutes have passed on the clock.
func stalled() turn { return turn{stall: true} }

// submit is one answer that submits. A files submission follows a read of the README: a Shape that
// inspected nothing cannot publish.
func submit(id string, args map[string]any) turn {
	if _, files := args["files"]; files {
		return answer(call(id+"-inspect", "read_file", map[string]any{"path": "README.md"}), call(id, "submit", args))
	}
	return answer(call(id, "submit", args))
}

// ack is the smallest submission that acknowledges accounted red: the feature, whole.
func ack(id string) turn {
	return submit(id, map[string]any{"files": []any{
		map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": feature},
	}})
}

// refinement is a submit_refinement call with the whole intent.md.
func refinement(id, path, content string) turn {
	return answer(call(id, "submit_refinement", map[string]any{"files": []any{
		map[string]any{"path": path, "content": content},
	}}))
}

// toolNamed is whether the request offers the tool, as name or function.name.
func toolNamed(req map[string]any, name string) bool {
	list, _ := req["tools"].([]any)
	for _, t := range list {
		m, _ := t.(map[string]any)
		if fn, ok := m["function"].(map[string]any); ok {
			m = fn
		}
		if m["name"] == name {
			return true
		}
	}
	return false
}

func (p *scripted) RoundTrip(r *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	var req map[string]any
	json.Unmarshal(body, &req)

	p.mu.Lock()
	n := p.count
	p.count++
	if d, ok := r.Context().Deadline(); ok {
		p.remaining = append(p.remaining, time.Until(d))
	} else {
		p.missing++
	}
	var t turn
	failed := false
	switch {
	case toolNamed(req, "submit_decision_check"):
		t = answer(call("decision-check", "submit_decision_check", map[string]any{"items": []any{}}))
	case toolNamed(req, "submit_literal_check"):
		t = answer(call("literal-check", "submit_literal_check", map[string]any{"items": []any{}}))
	case toolNamed(req, "submit_contract_review"):
		t = answer(call("contract-review", "submit_contract_review", map[string]any{"findings": []any{}, "environment": []any{}}))
	case toolNamed(req, "submit_contract_confirmation"):
		t = answer(call("contract-confirmation", "submit_contract_confirmation", map[string]any{"verdicts": []any{}}))
	default:
		p.shaper++
		if p.next < len(p.script) {
			t = p.script[p.next]
			p.next++
		} else {
			failed = true
		}
	}
	p.mu.Unlock()

	header := http.Header{}
	resp := &http.Response{StatusCode: 200, Header: header, Request: r, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1}
	switch {
	case failed:
		header.Set("Content-Type", "application/json")
		resp.StatusCode = 400
		resp.Status = "400 Bad Request"
		b, _ := json.Marshal(map[string]any{"error": map[string]any{"code": "invalid_request_error", "message": "scripted failure", "type": "invalid_request_error"}})
		resp.Body = io.NopCloser(bytes.NewReader(b))
	case t.stall:
		header.Set("Content-Type", "text/event-stream")
		resp.Status = "200 OK"
		resp.Body = &stallBody{clock: p.clock}
	default:
		header.Set("Content-Type", "text/event-stream")
		header.Set("X-Request-Id", "req-fixture")
		resp.Status = "200 OK"
		resp.Body = io.NopCloser(bytes.NewReader(wire(t, n)))
	}
	return resp, nil
}

// stallBody advances the client's clock past thirty minutes on the first read and then fails.
type stallBody struct {
	clock *clock
	once  sync.Once
}

func (b *stallBody) Read([]byte) (int, error) {
	b.once.Do(func() { b.clock.advance(30*time.Minute + time.Second) })
	return 0, io.ErrUnexpectedEOF
}

func (b *stallBody) Close() error { return nil }

// wire is the Responses event stream of a completed response with the items.
func wire(t turn, n int) []byte {
	var out bytes.Buffer
	frame := func(ev any) {
		b, _ := json.Marshal(ev)
		fmt.Fprintf(&out, "data: %s\n\n", b)
	}
	for i, it := range t.items {
		frame(map[string]any{"type": "response.output_item.done", "output_index": i, "item": it})
	}
	frame(map[string]any{"type": "response.completed", "response": map[string]any{
		"id": fmt.Sprintf("resp-%d", n+1), "model": "gpt-6.1-sol", "reasoning": map[string]any{"effort": "high"},
		"usage": map[string]any{
			"input_tokens": 100 + n, "input_tokens_details": map[string]any{"cached_tokens": 40, "cache_write_tokens": 0, "cache_write_tokens_details": map[string]any{"ttl_30m_tokens": 0, "ttl_12h_tokens": 0}},
			"output_tokens": 30, "output_tokens_details": map[string]any{"reasoning_tokens": 10},
		},
	}})
	return out.Bytes()
}

// observed is the time left to each request's deadline and how many requests had none.
func (p *scripted) observed() (remaining []time.Duration, missing int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]time.Duration(nil), p.remaining...), p.missing
}

// shaperRequests is how many requests took the script (the shaper's, not the reviews').
func (p *scripted) shaperRequests() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.shaper
}
