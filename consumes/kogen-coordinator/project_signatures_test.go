package consumes

import (
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
)

func TestProjectSignatures(t *testing.T) {
	const _ string = project.ConfigPath
	const _ string = project.DefaultProvider
	var (
		_ func(string, string) string                                                                         = project.ApprovalPath
		_ func(string) string                                                                                 = project.IntentDir
		_ func(string) string                                                                                 = project.FeaturePath
		_ func(string) string                                                                                 = project.IntentMDPath
		_ func(string) string                                                                                 = project.ModelPath
		_ func(string, string) string                                                                         = project.SourcePath
		_ func(string, string, *project.Config) ([]string, error)                                             = project.Baseline
		_ func(string, string) (*project.SourceRef, error)                                                    = project.BindSource
		_ func(string, *project.Config) (project.SourceOrigin, error)                                         = project.CaptureOrigin
		_ func(string, string, *project.Config, project.ShapeSource) error                                    = project.CaptureSource
		_ func(string, string, map[string][]byte) (*project.Intent, []project.Scenario, error)                = project.CheckIntent
		_ func(string) (project.CheckoutFacts, error)                                                         = project.Checkout
		_ func(project.Approval) ([]byte, error)                                                              = project.EncodeApproval
		_ func(any) ([]byte, error)                                                                           = project.EncodeJSON
		_ func(string, []string) string                                                                       = project.Hash
		_ func(string, string) bool                                                                           = project.IntentExists
		_ func(string) (*project.Config, error)                                                               = project.LoadConfig
		_ func(string, string) (*project.Intent, error)                                                       = project.LoadIntent
		_ func(string, string) project.SourceRequest                                                          = project.NewSourceRequest
		_ func(string) ([]project.Scenario, error)                                                            = project.ParseGherkin
		_ func(string) ([]project.Stage, error)                                                               = project.ParseStages
		_ func(string, *project.Config) ([]string, error)                                                     = project.Protected
		_ func(string, string) (*project.Approval, error)                                                     = project.ReadApproval
		_ func(string, string, map[string][]byte) (*project.Intent, error)                                    = project.ReadIntent
		_ func(string, string, project.SourceRequest, string, *project.SourceRef) (*project.SourceRef, error) = project.RefineSource
		_ func(string, string, string, *project.SourceRef) (*project.SourceRef, error)                        = project.RefreshSource
		_ func(string, string) error                                                                          = project.RequireIntentDir
		_ func(string, string, *project.Approval) (*project.SourceRef, error)                                 = project.ResolveApproval
		_ func(*project.SourceRef, []byte) error                                                              = project.RestoreSource
		_ func(*project.SourceRef) (string, error)                                                            = project.SourceCard
		_ func(string, string) (string, error)                                                                = project.SourceComparison
		_ func(*project.SourceRef) string                                                                     = project.SourceMessage
		_ func(*project.SourceRef) error                                                                      = project.ValidateSource
		_ func(string, string, *project.Config) error                                                         = project.WritePackageFacts

		_ func(*project.Config) bool                        = (*project.Config).AcceptanceConfigured
		_ func(*project.Intent) []string                    = (*project.Intent).StepFiles
		_ func(project.Role) bool                           = project.Role.External
		_ func(*project.SourceRef) *project.SourceRef       = (*project.SourceRef).Archived
		_ func(*project.SourceRef, *project.SourceRef) bool = (*project.SourceRef).Equal
		_ func(*project.SourceRef) map[string]any           = (*project.SourceRef).Map

		_ string                  = project.Acceptance{}.Run
		_ string                  = project.Acceptance{}.StepsDir
		_ string                  = project.Approval{}.Slug
		_ string                  = project.Approval{}.SHA256
		_ string                  = project.Approval{}.Base
		_ string                  = project.Approval{}.Approver
		_ string                  = project.Approval{}.ApprovedAt
		_ map[string]any          = project.Approval{}.SourceProvenance
		_ string                  = project.Check{}.Name
		_ string                  = project.Check{}.Run
		_ *project.Acceptance     = (&project.Config{}).Acceptance
		_ []project.Check         = (&project.Config{}).Checks
		_ []string                = (&project.Config{}).Fix
		_ map[string]project.Role = (&project.Config{}).Models
		_ []string                = (&project.Config{}).Protect
		_ []string                = (&project.Config{}).Setup
		_ []string                = (&project.Intent{}).Acceptance
		_ []string                = (&project.Intent{}).Decisions
		_ []string                = (&project.Intent{}).Feasibility
		_ *project.ModelSpec      = (&project.Intent{}).Model
		_ string                  = (&project.Intent{}).ModelLine
		_ string                  = (&project.Intent{}).Root
		_ string                  = (&project.Intent{}).Slug
		_ string                  = (&project.Intent{}).SlugLine
		_ string                  = (&project.Intent{}).Text
		_ string                  = (&project.Intent{}).Title
		_ []string                = (&project.Intent{}).Tests
		_ []string                = project.ModelSpec{}.Invariants
		_ bool                    = project.ModelSpec{}.None
		_ string                  = project.ModelSpec{}.Path
		_ string                  = project.ModelSpec{}.Samples
		_ string                  = project.ModelSpec{}.Seed
		_ string                  = project.ModelSpec{}.Steps
		_ string                  = project.Outline{}.Name
		_ string                  = project.Role{}.BaseURL
		_ string                  = project.Role{}.Effort
		_ string                  = project.Role{}.Executor
		_ string                  = project.Role{}.Model
		_ string                  = project.Role{}.Provider
		_ int                     = project.Scenario{}.Line
		_ string                  = project.Scenario{}.Name
		_ *project.Outline        = project.Scenario{}.Outline
		_ []project.Step          = project.Scenario{}.Steps
		_ project.SourceOrigin    = project.ShapeSource{}.Origin
		_ string                  = project.ShapeSource{}.Request
		_ []map[string]any        = project.ShapeSource{}.Rounds
		_ string                  = project.ShapeSource{}.SourcePath
		_ string                  = project.SourceOrigin{}.Configuration
		_ string                  = project.SourceOrigin{}.Head
		_ string                  = project.SourceOrigin{}.Tree
		_ string                  = project.SourceRef{}.ContractSHA256
		_ string                  = project.SourceRef{}.Path
		_ []string                = project.Stage{}.Checks
		_ string                  = project.Stage{}.Goal
		_ string                  = project.Stage{}.ID
		_ string                  = project.Stage{}.Title
		_ string                  = project.Step{}.Keyword
		_ int                     = project.Step{}.Line
		_ string                  = project.Step{}.Text
	)
}
