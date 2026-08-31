package lineage

import (
	"fmt"
	"strings"
)

func HumanReport(r Report) string {
	var builder strings.Builder
	builder.WriteString("# Gooo evaluator lineage report\n\n")
	fmt.Fprintf(&builder, "- decision: `%s`\n", r.Decision)
	fmt.Fprintf(&builder, "- case: `%s`\n", r.CaseID)
	fmt.Fprintf(&builder, "- reason: `%s`\n", r.Reason)
	fmt.Fprintf(&builder, "- parent_digest: `%s`\n", r.Summary.ParentDigest)
	fmt.Fprintf(&builder, "- child_digest: `%s`\n", r.Summary.ChildDigest)
	fmt.Fprintf(&builder, "- generation_depth: `%d -> %d`\n", r.Summary.Generation.ParentDepth, r.Summary.Generation.ChildDepth)
	fmt.Fprintf(&builder, "- inherited_counterexample_count: `%d`\n", r.Summary.InheritedCounterexampleCount)
	fmt.Fprintf(&builder, "- exact_cell_counts: `%s`\n", r.ExactSummary())
	fmt.Fprintf(&builder, "- improvement: `%s` (`%s`)\n", r.Improvement.State, r.Improvement.Reason)
	fmt.Fprintf(&builder, "- repository_writes: `%d`\n", r.Authority.RepositoryWrites)
	fmt.Fprintf(&builder, "- local_test_executions: `%d`\n", r.Authority.LocalTestExecutions)
	fmt.Fprintf(&builder, "- cross_project_required_gates: `%d`\n", r.Authority.CrossProjectRequiredGates)
	fmt.Fprintf(&builder, "- inventory_violations: `%d`\n", len(r.Inventory.Violations))
	builder.WriteString("\n## Authority chain\n\n")
	fmt.Fprintf(&builder, "`%s` (%s) -> `%s` (%s) -> `%s` (%s) -> `%s` (%s) -> `%s`\n", r.AuthorityChain.Source.Path, r.AuthorityChain.Source.Digest, r.AuthorityChain.SemanticIR.Path, r.AuthorityChain.SemanticIR.Digest, r.AuthorityChain.GeneratedGo.Path, r.AuthorityChain.GeneratedGo.Digest, r.AuthorityChain.Evaluator.Path, r.AuthorityChain.Evaluator.Digest, r.AuthorityChain.HumanReport.Path)
	if len(r.Unknowns) > 0 {
		builder.WriteString("\n## UNKNOWN records\n\n")
		for _, unknown := range r.Unknowns {
			fmt.Fprintf(&builder, "- stage=`%s`, step=`%s`, reason=`%s`, unknown_class=`%s`, next_operation=`%s`, blocked_by=`%s`\n", unknown.Stage, unknown.Step, unknown.Reason, unknown.UnknownClass, unknown.NextOperation, strings.Join(unknown.BlockedBy, ","))
		}
	}
	return builder.String()
}

func HumanConformanceSummary(reports []Report) string {
	var builder strings.Builder
	builder.WriteString("# Gooo evaluator lineage CI summary\n\n")
	builder.WriteString("| case | decision | parent digest | child digest | depth | inherited counterexamples | accepted | rejected | unknown | improvement |\n")
	builder.WriteString("|---|---|---|---|---:|---:|---:|---:|---:|---|\n")
	for _, report := range reports {
		fmt.Fprintf(&builder, "| %s | %s | `%s` | `%s` | %d -> %d | %d | %d | %d | %d | %s |\n", report.CaseID, report.Decision, report.Summary.ParentDigest, report.Summary.ChildDigest, report.Summary.Generation.ParentDepth, report.Summary.Generation.ChildDepth, report.Summary.InheritedCounterexampleCount, report.Summary.Cells.Accepted, report.Summary.Cells.Rejected, report.Summary.Cells.Unknown, report.Improvement.State)
	}
	builder.WriteString("\nFixed denominator: 12 cells. Precedence: REFUTED > UNKNOWN > CLOSED.\n\n")
	builder.WriteString("Runtime boundary: repository_writes=0, local_test_executions=0, cross_project_required_gates=0.\n")
	return builder.String()
}
