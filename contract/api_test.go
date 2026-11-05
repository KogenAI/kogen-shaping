package contract

import (
	"context"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/provider"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-shaping/shape"
	"github.com/KogenAI/kogen-verification/verify"
)

// The entry points kogen calls, with their types written out: a changed signature fails to
// compile.
var (
	_ func(context.Context, *sys.Session, shape.ShapeInput) (int, error)     = shape.Shape
	_ func(context.Context, *sys.Session, shape.RefineInput) (int, error)    = shape.Refine
	_ func(context.Context, *sys.Session, shape.ApproveInput) (int, error)   = shape.Approve
	_ func(context.Context, *sys.Session, shape.PreflightInput) (int, error) = shape.Preflight
)

// kogen's External is its optional external executor: the two types are assignable to each other.
var (
	_ func(project.Role) (executor.Executor, error) = shape.External(nil)
	_ shape.External                                = func(project.Role) (executor.Executor, error) { return nil, nil }
)

// inputs sets every field kogen sets, each from a variable of the field's type: a changed name or
// type fails to compile.
func inputs(
	projectDir, slug, request, requestPath, installRoot, hash, by string,
	tools verify.Tools, client *provider.Client, external shape.External, jsonMode bool, checkout any,
) (shape.ShapeInput, shape.RefineInput, shape.ApproveInput, shape.PreflightInput) {
	return shape.ShapeInput{Project: projectDir, Slug: slug, Request: request, RequestPath: requestPath, Tools: tools, InstallRoot: installRoot, Client: client, External: external},
		shape.RefineInput{Project: projectDir, Slug: slug, Hash: hash, Request: request, RequestPath: requestPath, Tools: tools, InstallRoot: installRoot, Client: client, External: external},
		shape.ApproveInput{Project: projectDir, Slug: slug, Hash: hash, By: by, Tools: tools},
		shape.PreflightInput{Project: projectDir, Slug: slug, JSON: jsonMode, Checkout: checkout}
}

// TestInputFieldsKeepTheirMeaning holds that the fields kogen sets arrive as given.
func TestInputFieldsKeepTheirMeaning(t *testing.T) {
	t.Parallel()
	client := &provider.Client{}
	var tools verify.Tools
	checkout := map[string]any{"id": "x"}
	sh, rf, ap, pf := inputs("p", "s", "r", "/rp", "/ir", "h", "b", tools, client, nil, true, checkout)
	if sh.Project != "p" || sh.Slug != "s" || sh.Request != "r" || sh.RequestPath != "/rp" || sh.InstallRoot != "/ir" || sh.Client != client || sh.External != nil {
		t.Fatalf("ShapeInput %+v", sh)
	}
	if rf.Project != "p" || rf.Slug != "s" || rf.Hash != "h" || rf.Request != "r" || rf.RequestPath != "/rp" || rf.InstallRoot != "/ir" || rf.Client != client || rf.External != nil {
		t.Fatalf("RefineInput %+v", rf)
	}
	if ap.Project != "p" || ap.Slug != "s" || ap.Hash != "h" || ap.By != "b" {
		t.Fatalf("ApproveInput %+v", ap)
	}
	if pf.Project != "p" || pf.Slug != "s" || !pf.JSON || pf.Checkout == nil {
		t.Fatalf("PreflightInput %+v", pf)
	}
}
