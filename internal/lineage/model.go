package lineage

import "encoding/json"

const (
	Schema         = "gooo/evaluator-lineage/report/v1"
	InputSchema    = "gooo/evaluator-lineage/input/v1"
	IRSchema       = "gooo/evaluator-lineage/ir/v1"
	ContractSchema = "gooo/evaluator-lineage/denominator/v1"
	StateClosed    = "CLOSED"
	StateUnknown   = "UNKNOWN"
	StateRefuted   = "REFUTED"
)

var Precedence = []string{StateRefuted, StateUnknown, StateClosed}

type Input struct {
	Schema         string            `json:"schema"`
	CaseID         string            `json:"case_id"`
	Parent         *ReleaseReference `json:"parent"`
	Child          ReleaseReference  `json:"child"`
	Cells          []CellEvidence    `json:"cells"`
	UpperDecision  string            `json:"upper_decision"`
	Improvement    *ImprovementInput `json:"improvement"`
	Authority      AuthorityInput    `json:"authority"`
}

type ReleaseReference struct {
	ID                        string `json:"id"`
	Version                   string `json:"version"`
	ReleaseTag                string `json:"release_tag"`
	ReleaseDigest             string `json:"release_digest"`
	ObservedReleaseDigest     string `json:"observed_release_digest"`
	GenerationDepth           int    `json:"generation_depth"`
	InheritedCounterexampleCount int  `json:"inherited_counterexample_count"`
	Decision                  string `json:"decision"`
	SelfAttestation           *SelfAttestation `json:"self_attestation"`
}

type SelfAttestation struct {
	EvaluatorID    string `json:"evaluator_id"`
	ReleaseDigest  string `json:"release_digest"`
	Decision       string `json:"decision"`
}

type CellEvidence struct {
	ID             string `json:"id"`
	Decision       string `json:"decision"`
	EvidenceDigest string `json:"evidence_digest"`
}

type ImprovementInput struct {
	MetricID    string      `json:"metric_id"`
	InputDigest string      `json:"input_digest"`
	Before      *ExactValue `json:"before"`
	After       *ExactValue `json:"after"`
}

type ExactValue struct {
	Value  string `json:"value"`
	Digest string `json:"digest"`
}

type AuthorityInput struct {
	RepositoryWrites        int `json:"repository_writes"`
	LocalTestExecutions     int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
}

type SemanticIR struct {
	Schema       string       `json:"schema"`
	SourcePath   string       `json:"source_path"`
	SourceDigest string       `json:"source_digest"`
	Nodes        []ActivityIR `json:"nodes"`
}

type ActivityIR struct {
	ID             string `json:"id"`
	Activity       string `json:"activity"`
	Name           string `json:"name"`
	ProofChoice    string `json:"proof_choice"`
	IndicatorClass string `json:"indicator_class"`
	MetricID       string `json:"metric_id"`
	MetricPath     string `json:"metric_path"`
	Artifact       string `json:"artifact"`
	Evaluator      string `json:"evaluator"`
	SourceLine     int    `json:"source_line"`
}

type Denominator struct {
	Schema           string             `json:"schema"`
	DenominatorID    string             `json:"denominator_id"`
	CandidateID      string             `json:"candidate_id"`
	Total            int                `json:"total"`
	Proofs           []Balance           `json:"proofs"`
	IndicatorClasses []Balance           `json:"indicator_classes"`
	Cells            []DenominatorCell  `json:"cells"`
}

type Balance struct {
	Choice string `json:"choice,omitempty"`
	Class  string `json:"class,omitempty"`
	Total  int    `json:"total"`
}

type DenominatorCell struct {
	Ordinal        int    `json:"ordinal"`
	ID             string `json:"id"`
	Activity       string `json:"activity"`
	ProofChoice    string `json:"proof_choice"`
	IndicatorClass string `json:"indicator_class"`
	MetricID       string `json:"metric_id"`
	MetricPath     string `json:"metric_path"`
	Artifact       string `json:"artifact"`
	Evaluator      string `json:"evaluator"`
}

type Unknown struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type CellReport struct {
	ID             string `json:"id"`
	Activity       string `json:"activity"`
	MetricID       string `json:"metric_id"`
	ProofChoice    string `json:"proof_choice"`
	IndicatorClass string `json:"indicator_class"`
	Decision       string `json:"decision"`
	EvidenceDigest string `json:"evidence_digest"`
}

type Counts struct {
	Total    int `json:"total"`
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
	Unknown  int `json:"unknown"`
}

type Generation struct {
	ParentDepth int `json:"parent_depth"`
	ChildDepth  int `json:"child_depth"`
}

type Summary struct {
	DenominatorTotal                 int        `json:"denominator_total"`
	ParentDigest                    string     `json:"parent_digest"`
	ChildDigest                     string     `json:"child_digest"`
	Generation                      Generation `json:"generation_depth"`
	InheritedCounterexampleCount    int        `json:"inherited_counterexample_count"`
	Cells                           Counts     `json:"cells"`
}

type Meta struct {
	SourcePath       string
	SourceDigest     string
	SemanticIRPath   string
	SemanticIRDigest string
	GeneratedGoPath  string
	GeneratedGoDigest string
	EvaluatorPath    string
	EvaluatorDigest  string
	ContractPath     string
	ContractDigest   string
	HumanReportPath  string
	Denominator      Denominator
}

type ImprovementReport struct {
	State       string      `json:"state"`
	MetricID    string      `json:"metric_id"`
	Reason      string      `json:"reason"`
	Before      *ExactValue `json:"before"`
	After       *ExactValue `json:"after"`
}

type AuthorityReport struct {
	RepositoryWrites          int `json:"repository_writes"`
	LocalTestExecutions       int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
	RequestedRepositoryWrites int `json:"requested_repository_writes"`
	RequestedLocalTestExecutions int `json:"requested_local_test_executions"`
	RequestedCrossProjectRequiredGates int `json:"requested_cross_project_required_gates"`
	ReadOnly                  bool `json:"read_only"`
}

type AuthorityArtifact struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type AuthorityChain struct {
	Source      AuthorityArtifact `json:"source"`
	SemanticIR  AuthorityArtifact `json:"semantic_ir"`
	GeneratedGo AuthorityArtifact `json:"generated_go"`
	Evaluator   AuthorityArtifact `json:"evaluator"`
	HumanReport AuthorityArtifact `json:"human_report"`
}

type Inventory struct {
	Violations []string `json:"violations"`
	Exclusions []string `json:"exclusions"`
}

type Report struct {
	Schema             string            `json:"schema"`
	Decision           string            `json:"decision"`
	CaseID             string            `json:"case_id"`
	InputDigest        string            `json:"input_digest"`
	Precedence         []string          `json:"precedence"`
	Summary            Summary           `json:"summary"`
	Cells              []CellReport      `json:"cells"`
	Unknowns           []Unknown         `json:"unknowns"`
	Improvement        ImprovementReport `json:"improvement"`
	Authority          AuthorityReport   `json:"authority"`
	AuthorityChain     AuthorityChain    `json:"authority_chain"`
	Inventory          Inventory         `json:"inventory"`
	Reason             string            `json:"reason"`
}

func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
