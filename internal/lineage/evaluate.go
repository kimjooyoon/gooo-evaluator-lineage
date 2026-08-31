package lineage

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kimjooyoon/gooo-evaluator-lineage/internal/generated"
)

func Evaluate(raw []byte, meta Meta) Report {
	report := baseReport(meta, DigestBytes(raw))
	var input Input
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return refute(report, "MALFORMED_INPUT", "REPAIR_INPUT")
	}
	report.CaseID = input.CaseID
	report.Summary = summaryFor(input)
	report.Authority = authorityFor(input.Authority)

	hasRefuted := false
	refutedReason := ""
	addRefutation := func(reason string) {
		if !hasRefuted {
			refutedReason = reason
		}
		hasRefuted = true
	}
	unknownReason := ""
	addUnknownReason := func(reason string) {
		if unknownReason == "" {
			unknownReason = reason
		}
	}

	if input.Schema != InputSchema || input.CaseID == "" {
		addRefutation("INVALID_INPUT_CONTRACT")
	}
	if err := validateMeta(meta); err != nil {
		addRefutation(err.Error())
	}
	if err := ValidateDenominator(meta.Denominator); err != nil {
		addRefutation(err.Error())
	}
	if input.Authority.RepositoryWrites != 0 || input.Authority.LocalTestExecutions != 0 || input.Authority.CrossProjectRequiredGates != 0 {
		addRefutation("AUTHORITY_ESCALATION_REFUTED")
	}

	cellByID := make(map[string]DenominatorCell, len(meta.Denominator.Cells))
	for _, cell := range meta.Denominator.Cells {
		cellByID[cell.ID] = cell
	}
	seenCellIDs := map[string]bool{}
	for _, evidence := range input.Cells {
		cell, bound := cellByID[evidence.ID]
		cellReport := CellReport{
			ID:             evidence.ID,
			Decision:       evidence.Decision,
			EvidenceDigest: evidence.EvidenceDigest,
			Activity:       "UNBOUND",
			MetricID:       "UNBOUND",
		}
		if bound {
			cellReport.Activity = cell.Activity
			cellReport.MetricID = cell.MetricID
			cellReport.ProofChoice = cell.ProofChoice
			cellReport.IndicatorClass = cell.IndicatorClass
		} else {
			addRefutation("CELL_NOT_IN_FIXED_DENOMINATOR")
		}
		if seenCellIDs[evidence.ID] {
			addRefutation("DUPLICATE_CELL_EVIDENCE")
		}
		seenCellIDs[evidence.ID] = true
		switch evidence.Decision {
		case "ACCEPTED":
		case "REJECTED":
			addRefutation("EXPLICIT_CONTRADICTION")
		case "UNKNOWN":
			addUnknownReason("CELL_DECISION_UNKNOWN")
			addUnknown(&report, Unknown{
				Stage:         "CELL_EVIDENCE",
				Step:          "OBSERVE_CELL_DECISION",
				Reason:        "CELL_DECISION_UNKNOWN",
				UnknownClass:  "EVIDENCE_UNAVAILABLE",
				NextOperation: "PROVIDE_CELL_EVIDENCE",
				BlockedBy:     []string{"cell:" + evidence.ID},
			})
		default:
			addRefutation("INVALID_CELL_DECISION")
		}
		if evidence.EvidenceDigest != "" && !validDigest(evidence.EvidenceDigest) {
			addRefutation("INVALID_CELL_EVIDENCE_DIGEST")
		}
	}
	if len(input.Cells) != meta.Denominator.Total {
		addRefutation("FIXED_DENOMINATOR_SHRINK_REFUTED")
	}
	for _, cell := range meta.Denominator.Cells {
		if !seenCellIDs[cell.ID] {
			addRefutation("FIXED_DENOMINATOR_CELL_MISSING")
		}
	}

	childValid := validateReference(input.Child, false, &report, addRefutation)
	parentValid := false
	if input.Parent == nil {
		addUnknownReason("PARENT_REFERENCE_MISSING")
		addUnknown(&report, Unknown{
			Stage:         "PARENT",
			Step:          "LOAD_IMMUTABLE_PARENT_RELEASE",
			Reason:        "PARENT_REFERENCE_MISSING",
			UnknownClass:  "DEPENDENCY_UNAVAILABLE",
			NextOperation: "PROVIDE_IMMUTABLE_PARENT_RELEASE_DIGEST",
			BlockedBy:     []string{"parent-release"},
		})
	} else {
		parentValid = validateReference(*input.Parent, true, &report, addRefutation)
	}
	if parentValid && childValid {
		parent := *input.Parent
		child := input.Child
		report.Summary.ParentDigest = parent.ReleaseDigest
		report.Summary.ChildDigest = child.ReleaseDigest
		report.Summary.Generation = Generation{ParentDepth: parent.GenerationDepth, ChildDepth: child.GenerationDepth}
		report.Summary.InheritedCounterexampleCount = child.InheritedCounterexampleCount
		if parent.ID == child.ID || parent.ReleaseDigest == child.ReleaseDigest {
			addRefutation("PARENT_CHILD_IDENTITY_COLLISION")
		}
		if child.GenerationDepth != parent.GenerationDepth+1 {
			addRefutation("GENERATION_DEPTH_NOT_PARENT_PLUS_ONE")
		}
		if child.InheritedCounterexampleCount < parent.InheritedCounterexampleCount {
			addRefutation("INHERITED_COUNTEREXAMPLE_COUNT_DECREASED")
		}
		if parent.ObservedReleaseDigest != parent.ReleaseDigest {
			addUnknownReason("STALE_PARENT_RELEASE")
			addUnknown(&report, Unknown{
				Stage:         "PARENT",
				Step:          "VERIFY_PARENT_RELEASE_DIGEST",
				Reason:        "STALE_PARENT_RELEASE",
				UnknownClass:  "STALE_IMMUTABLE_RELEASE",
				NextOperation: "PIN_CURRENT_PARENT_RELEASE_DIGEST",
				BlockedBy:     []string{"parent-release-digest"},
			})
		}
		if child.ObservedReleaseDigest != child.ReleaseDigest {
			addUnknownReason("STALE_CHILD_RELEASE")
			addUnknown(&report, Unknown{
				Stage:         "CHILD",
				Step:          "VERIFY_CHILD_RELEASE_DIGEST",
				Reason:        "STALE_CHILD_RELEASE",
				UnknownClass:  "STALE_IMMUTABLE_RELEASE",
				NextOperation: "PIN_CURRENT_CHILD_RELEASE_DIGEST",
				BlockedBy:     []string{"child-release-digest"},
			})
		}
		if child.SelfAttestation != nil && (child.SelfAttestation.EvaluatorID == child.ID || child.SelfAttestation.ReleaseDigest == child.ReleaseDigest) {
			addRefutation("CHILD_SELF_ATTESTATION_REFUTED")
		}
		if input.UpperDecision != parent.Decision {
			if input.UpperDecision == StateUnknown || parent.Decision == StateUnknown {
				addUnknownReason("UNKNOWN_UPPER_DECISION")
				addUnknown(&report, Unknown{
					Stage:         "PARENT",
					Step:          "READ_UPPER_DECISION",
					Reason:        "UNKNOWN_UPPER_DECISION",
					UnknownClass:  "UPPER_DECISION_UNAVAILABLE",
					NextOperation: "PROVIDE_CLOSED_OR_REFUTED_PARENT_DECISION",
					BlockedBy:     []string{"upper-decision"},
				})
			} else {
				addRefutation("UPPER_DECISION_MISMATCH")
			}
		}
		if input.UpperDecision == StateUnknown || parent.Decision == StateUnknown {
			addUnknownReason("UNKNOWN_UPPER_DECISION")
			if !hasUnknownReason(report, "UNKNOWN_UPPER_DECISION") {
				addUnknown(&report, Unknown{
					Stage:         "PARENT",
					Step:          "READ_UPPER_DECISION",
					Reason:        "UNKNOWN_UPPER_DECISION",
					UnknownClass:  "UPPER_DECISION_UNAVAILABLE",
					NextOperation: "PROVIDE_CLOSED_OR_REFUTED_PARENT_DECISION",
					BlockedBy:     []string{"upper-decision"},
				})
			}
		}
		if input.UpperDecision == StateRefuted || parent.Decision == StateRefuted {
			addRefutation("PARENT_DECISION_REFUTED")
		}
	}

	report.Improvement = improvementFor(input, meta.Denominator)
	if report.Improvement.State == StateUnknown {
		addUnknownReason("EXACT_BEFORE_AFTER_PAIR_MISSING")
		addUnknown(&report, Unknown{
			Stage:         "IMPROVEMENT",
			Step:          "REQUIRE_EXACT_BEFORE_AFTER_PAIR",
			Reason:        report.Improvement.Reason,
			UnknownClass:  "CAUSALITY_UNPROVEN",
			NextOperation: "PROVIDE_EXACT_BEFORE_AFTER_PAIR",
			BlockedBy:     []string{"exact-before-after-pair"},
		})
	}

	report.Decision = resolveState(hasRefuted, len(report.Unknowns))
	if report.Decision == StateRefuted {
		report.Reason = refutedReason
	} else if report.Decision == StateUnknown {
		report.Reason = unknownReason
	} else {
		report.Reason = "PARENT_RELEASE_ACCEPTED_CHILD_CANDIDATE"
	}
	return report
}

func resolveState(hasRefuted bool, unknownCount int) string {
	if hasRefuted {
		return StateRefuted
	}
	if unknownCount > 0 {
		return StateUnknown
	}
	return StateClosed
}

func baseReport(meta Meta, inputDigest string) Report {
	return Report{
		Schema:      Schema,
		Decision:    StateRefuted,
		InputDigest: inputDigest,
		Precedence:  append([]string(nil), Precedence...),
		Summary: Summary{DenominatorTotal: 12, Cells: Counts{Total: 0}},
		Cells:       []CellReport{},
		Unknowns:    []Unknown{},
		AuthorityChain: AuthorityChain{
			Source:      AuthorityArtifact{Path: meta.SourcePath, Digest: meta.SourceDigest},
			SemanticIR:  AuthorityArtifact{Path: meta.SemanticIRPath, Digest: meta.SemanticIRDigest},
			GeneratedGo: AuthorityArtifact{Path: meta.GeneratedGoPath, Digest: meta.GeneratedGoDigest},
			Evaluator:   AuthorityArtifact{Path: meta.EvaluatorPath, Digest: meta.EvaluatorDigest},
			HumanReport: AuthorityArtifact{Path: meta.HumanReportPath},
		},
		Inventory: Inventory{Violations: []string{}, Exclusions: []string{"PROJECT_ROOT_README"}},
	}
}

func summaryFor(input Input) Summary {
	counts := Counts{Total: len(input.Cells)}
	for _, cell := range input.Cells {
		switch cell.Decision {
		case "ACCEPTED":
			counts.Accepted++
		case "REJECTED":
			counts.Rejected++
		case "UNKNOWN":
			counts.Unknown++
		}
	}
	summary := Summary{DenominatorTotal: 12, Cells: counts}
	if input.Parent != nil {
		summary.ParentDigest = input.Parent.ReleaseDigest
		summary.Generation.ParentDepth = input.Parent.GenerationDepth
	}
	summary.ChildDigest = input.Child.ReleaseDigest
	summary.Generation.ChildDepth = input.Child.GenerationDepth
	summary.InheritedCounterexampleCount = input.Child.InheritedCounterexampleCount
	return summary
}

func authorityFor(input AuthorityInput) AuthorityReport {
	return AuthorityReport{
		RepositoryWrites:                    0,
		LocalTestExecutions:                 0,
		CrossProjectRequiredGates:            0,
		RequestedRepositoryWrites:            input.RepositoryWrites,
		RequestedLocalTestExecutions:         input.LocalTestExecutions,
		RequestedCrossProjectRequiredGates:   input.CrossProjectRequiredGates,
		ReadOnly: input.RepositoryWrites == 0 && input.LocalTestExecutions == 0 && input.CrossProjectRequiredGates == 0,
	}
}

func validateMeta(meta Meta) error {
	if meta.SourcePath == "" || !validDigest(meta.SourceDigest) || meta.SemanticIRPath == "" || !validDigest(meta.SemanticIRDigest) || meta.GeneratedGoPath == "" || !validDigest(meta.GeneratedGoDigest) || meta.EvaluatorPath == "" || !validDigest(meta.EvaluatorDigest) || meta.ContractPath == "" || !validDigest(meta.ContractDigest) {
		return errors.New("INCOMPLETE_AUTHORITY_CHAIN")
	}
	if generated.SourcePath != meta.SourcePath || generated.SourceDigest != meta.SourceDigest || generated.SemanticIRDigest != meta.SemanticIRDigest || generated.ContractPath != meta.ContractPath || generated.ContractDigest != meta.ContractDigest || generated.ActivityCount != 12 || len(generated.Activities) != 12 {
		return errors.New("GENERATED_AUTHORITY_CHAIN_MISMATCH")
	}
	for index, cell := range meta.Denominator.Cells {
		activity := generated.Activities[index]
		if activity.ID != activityIDForCell(cell.ID) || activity.Activity != cell.Activity || activity.MetricID != cell.MetricID || activity.MetricPath != cell.MetricPath || activity.ProofChoice != cell.ProofChoice || activity.IndicatorClass != cell.IndicatorClass || activity.Artifact != cell.Artifact || activity.Evaluator != cell.Evaluator {
			return errors.New("GENERATED_ACTIVITY_BINDING_MISMATCH")
		}
	}
	return nil
}

func validateReference(reference ReleaseReference, parent bool, report *Report, addRefutation func(string)) bool {
	if reference.ID == "" || !validVersion(reference.Version) || reference.ReleaseTag != reference.Version || !validDigest(reference.ReleaseDigest) || !validDigest(reference.ObservedReleaseDigest) || reference.GenerationDepth < 0 || reference.InheritedCounterexampleCount < 0 {
		addRefutation("INVALID_IMMUTABLE_RELEASE_REFERENCE")
		return false
	}
	if parent {
		if reference.Decision != StateClosed && reference.Decision != StateUnknown && reference.Decision != StateRefuted {
			addRefutation("INVALID_PARENT_DECISION")
			return false
		}
	} else if reference.Decision != "" {
		addRefutation("CHILD_CANDIDATE_MUST_NOT_ASSERT_DECISION")
		return false
	}
	if reference.SelfAttestation != nil && reference.SelfAttestation.EvaluatorID == "" {
		addRefutation("INVALID_SELF_ATTESTATION")
		return false
	}
	if report.Summary.ChildDigest == "" && !parent {
		report.Summary.ChildDigest = reference.ReleaseDigest
	}
	return true
}

func improvementFor(input Input, denominator Denominator) ImprovementReport {
	result := ImprovementReport{State: StateUnknown, Reason: "EXACT_BEFORE_AFTER_PAIR_MISSING"}
	if input.Improvement == nil {
		return result
	}
	result.MetricID = input.Improvement.MetricID
	result.Before = input.Improvement.Before
	result.After = input.Improvement.After
	if input.Improvement.Before == nil || input.Improvement.After == nil || input.Improvement.Before.Value == "" || input.Improvement.After.Value == "" {
		if input.Improvement.Before != nil && input.Improvement.After != nil {
			result.Reason = "EXACT_BEFORE_AFTER_VALUE_MISSING"
		}
		return result
	}
	if !validDigest(input.Improvement.InputDigest) || !validDigest(input.Improvement.Before.Digest) || !validDigest(input.Improvement.After.Digest) || !metricInDenominator(input.Improvement.MetricID, denominator) {
		result.Reason = "EXACT_BEFORE_AFTER_PAIR_NOT_BOUND"
		return result
	}
	result.State = StateClosed
	result.Reason = "EXACT_BEFORE_AFTER_PAIR_PRESENT"
	return result
}

func metricInDenominator(metricID string, denominator Denominator) bool {
	for _, cell := range denominator.Cells {
		if cell.MetricID == metricID {
			return true
		}
	}
	return false
}

func addUnknown(report *Report, unknown Unknown) {
	if unknown.Stage == "" || unknown.Step == "" || unknown.Reason == "" || unknown.UnknownClass == "" || unknown.NextOperation == "" || len(unknown.BlockedBy) == 0 {
		return
	}
	for _, existing := range report.Unknowns {
		if existing.Stage == unknown.Stage && existing.Step == unknown.Step && existing.Reason == unknown.Reason && existing.UnknownClass == unknown.UnknownClass && existing.NextOperation == unknown.NextOperation {
			return
		}
	}
	report.Unknowns = append(report.Unknowns, unknown)
}

func hasUnknownReason(report Report, reason string) bool {
	for _, unknown := range report.Unknowns {
		if unknown.Reason == reason {
			return true
		}
	}
	return false
}

func refute(report Report, reason string, nextOperation string) Report {
	report.Decision = StateRefuted
	report.Reason = reason
	report.Unknowns = []Unknown{}
	if nextOperation == "" {
		nextOperation = "REPAIR_INPUT"
	}
	return report
}

func validVersion(value string) bool {
	if len(value) < 6 || value[0] != 'v' {
		return false
	}
	parts := strings.Split(value[1:], ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, character := range part {
			if character < '0' || character > '9' {
				return false
			}
		}
	}
	return true
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func (r Report) ValidateUnknowns() error {
	for _, unknown := range r.Unknowns {
		if unknown.Stage == "" || unknown.Step == "" || unknown.Reason == "" || unknown.UnknownClass == "" || unknown.NextOperation == "" || len(unknown.BlockedBy) == 0 {
			return errors.New("UNKNOWN_RECORD_MISSING_REQUIRED_FIELD")
		}
	}
	return nil
}

func (r Report) ExactSummary() string {
	return fmt.Sprintf("accepted=%d rejected=%d unknown=%d total=%d", r.Summary.Cells.Accepted, r.Summary.Cells.Rejected, r.Summary.Cells.Unknown, r.Summary.Cells.Total)
}
