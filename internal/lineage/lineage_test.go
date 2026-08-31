package lineage

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolutionPrecedence(t *testing.T) {
	if got := resolveState(false, 0); got != StateClosed {
		t.Fatalf("closed state = %s", got)
	}
	if got := resolveState(false, 1); got != StateUnknown {
		t.Fatalf("unknown state = %s", got)
	}
	if got := resolveState(true, 1); got != StateRefuted {
		t.Fatalf("refuted state = %s", got)
	}
}

func TestUnknownRecordHasAllSixFields(t *testing.T) {
	report := Report{Unknowns: []Unknown{{
		Stage:         "PARENT",
		Step:          "LOAD_IMMUTABLE_PARENT_RELEASE",
		Reason:        "PARENT_REFERENCE_MISSING",
		UnknownClass:  "DEPENDENCY_UNAVAILABLE",
		NextOperation: "PROVIDE_IMMUTABLE_PARENT_RELEASE_DIGEST",
		BlockedBy:     []string{"parent-release"},
	}}}
	if err := report.ValidateUnknowns(); err != nil {
		t.Fatal(err)
	}
}

func TestSourceAndDenominatorCompileToTwelveActivities(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	source, err := os.ReadFile(filepath.Join(root, "examples/evaluator-lineage/main.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	denominator, _, err := LoadDenominator(filepath.Join(root, "contracts/evaluator-lineage-denominator-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	ir, err := CompileSource("examples/evaluator-lineage/main.gooo", source, denominator)
	if err != nil {
		t.Fatal(err)
	}
	if len(ir.Nodes) != 12 {
		t.Fatalf("activity count = %d", len(ir.Nodes))
	}
}

func TestReleaseReferencesRejectMutableVersionShape(t *testing.T) {
	if validVersion("main") || validVersion("pr-42") || validVersion("v0.1") {
		t.Fatal("mutable or incomplete version accepted")
	}
	if !validVersion("v0.1.0") {
		t.Fatal("immutable release version rejected")
	}
}
