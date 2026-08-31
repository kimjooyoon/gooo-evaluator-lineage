package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-evaluator-lineage/internal/lineage"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "compile":
		return compile(args[1:], stdout, stderr)
	case "evaluate":
		return evaluate(args[1:], stdout, stderr)
	case "conformance":
		return conformance(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, "gooo-evaluator-lineage/v0.1.0")
		return 0
	default:
		usage(stderr)
		return 2
	}
}

func compile(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("compile", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sourcePath := flags.String("source", "examples/evaluator-lineage/main.gooo", "Gooo source")
	contractPath := flags.String("contract", "contracts/evaluator-lineage-denominator-v1.json", "fixed denominator")
	outputIR := flags.String("output-ir", "internal/generated/semantic-ir.json", "semantic IR output")
	outputGo := flags.String("output-go", "internal/generated/semantic.gooo.go", "generated Go output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	source, err := os.ReadFile(*sourcePath)
	if err != nil {
		fmt.Fprintf(stderr, "read source: %v\n", err)
		return 1
	}
	denominator, contract, err := lineage.LoadDenominator(*contractPath)
	if err != nil {
		fmt.Fprintf(stderr, "read denominator: %v\n", err)
		return 1
	}
	ir, err := lineage.CompileSource(*sourcePath, source, denominator)
	if err != nil {
		fmt.Fprintf(stderr, "compile source: %v\n", err)
		return 1
	}
	irBytes, err := lineage.SemanticIRBytes(ir)
	if err != nil {
		fmt.Fprintf(stderr, "encode semantic IR: %v\n", err)
		return 1
	}
	goBytes, err := lineage.GenerateGo(ir, lineage.DigestBytes(irBytes), lineage.DigestBytes(contract))
	if err != nil {
		fmt.Fprintf(stderr, "generate Go: %v\n", err)
		return 1
	}
	if err := writeFile(*outputIR, irBytes); err != nil {
		fmt.Fprintf(stderr, "write semantic IR: %v\n", err)
		return 1
	}
	if err := writeFile(*outputGo, goBytes); err != nil {
		fmt.Fprintf(stderr, "write generated Go: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "compiled source=%s semantic_ir=%s generated_go=%s\n", *sourcePath, *outputIR, *outputGo)
	return 0
}

func evaluate(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	inputPath := flags.String("input", "", "lineage input JSON")
	outputDir := flags.String("output-dir", "artifacts", "report directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *inputPath == "" {
		fmt.Fprintln(stderr, "evaluate requires -input")
		return 2
	}
	report, err := evaluateFile(*inputPath, *outputDir, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "evaluate: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%s decision=%s %s\n", report.CaseID, report.Decision, report.ExactSummary())
	return 0
}

func conformance(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	flags.SetOutput(stderr)
	fixtureDir := flags.String("fixtures", "fixtures", "fixture directory")
	outputDir := flags.String("output-dir", "artifacts/conformance", "report directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	entries, err := os.ReadDir(*fixtureDir)
	if err != nil {
		fmt.Fprintf(stderr, "read fixtures: %v\n", err)
		return 1
	}
	var reports []lineage.Report
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		inputPath := filepath.Join(*fixtureDir, entry.Name())
		report, evalErr := evaluateFile(inputPath, filepath.Join(*outputDir, strings.TrimSuffix(entry.Name(), ".json")), stderr)
		if evalErr != nil {
			fmt.Fprintf(stderr, "%s: %v\n", entry.Name(), evalErr)
			return 1
		}
		expected, ok := expectedDecision(strings.TrimSuffix(entry.Name(), ".json"))
		if !ok || report.Decision != expected {
			fmt.Fprintf(stderr, "%s: expected decision %s, got %s\n", entry.Name(), expected, report.Decision)
			return 1
		}
		if err := report.ValidateUnknowns(); err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", entry.Name(), err)
			return 1
		}
		reports = append(reports, report)
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].CaseID < reports[j].CaseID })
	summary := lineage.HumanConformanceSummary(reports)
	if err := writeFile(filepath.Join(*outputDir, "ci-summary.md"), []byte(summary)); err != nil {
		fmt.Fprintf(stderr, "write CI summary: %v\n", err)
		return 1
	}
	fmt.Fprint(stdout, summary)
	return 0
}

func evaluateFile(inputPath string, outputDir string, stderr io.Writer) (lineage.Report, error) {
	input, err := os.ReadFile(inputPath)
	if err != nil {
		return lineage.Report{}, err
	}
	sourcePath := "examples/evaluator-lineage/main.gooo"
	contractPath := "contracts/evaluator-lineage-denominator-v1.json"
	semanticPath := "internal/generated/semantic-ir.json"
	generatedPath := "internal/generated/semantic.gooo.go"
	evaluatorPath := "internal/lineage/evaluate.go"
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return lineage.Report{}, err
	}
	contract, err := os.ReadFile(contractPath)
	if err != nil {
		return lineage.Report{}, err
	}
	semantic, err := os.ReadFile(semanticPath)
	if err != nil {
		return lineage.Report{}, err
	}
	if err := validateSemanticIR(semantic); err != nil {
		return lineage.Report{}, err
	}
	generatedGo, err := os.ReadFile(generatedPath)
	if err != nil {
		return lineage.Report{}, err
	}
	evaluator, err := os.ReadFile(evaluatorPath)
	if err != nil {
		return lineage.Report{}, err
	}
	denominator, _, err := lineage.LoadDenominator(contractPath)
	if err != nil {
		return lineage.Report{}, err
	}
	reportPath := filepath.Join(outputDir, "report.json")
	humanPath := filepath.Join(outputDir, "report.md")
	meta := lineage.Meta{
		SourcePath:        sourcePath,
		SourceDigest:      lineage.DigestBytes(source),
		SemanticIRPath:    semanticPath,
		SemanticIRDigest:  lineage.DigestBytes(semantic),
		GeneratedGoPath:   generatedPath,
		GeneratedGoDigest: lineage.DigestBytes(generatedGo),
		EvaluatorPath:     evaluatorPath,
		EvaluatorDigest:   lineage.DigestBytes(evaluator),
		ContractPath:      contractPath,
		ContractDigest:    lineage.DigestBytes(contract),
		HumanReportPath:   humanPath,
		Denominator:       denominator,
	}
	report := lineage.Evaluate(input, meta)
	human := lineage.HumanReport(report)
	report.AuthorityChain.HumanReport.Digest = lineage.DigestBytes([]byte(human))
	if err := writeReport(reportPath, report); err != nil {
		return lineage.Report{}, err
	}
	if err := writeFile(humanPath, []byte(human)); err != nil {
		return lineage.Report{}, err
	}
	return report, nil
}

func validateSemanticIR(raw []byte) error {
	var ir lineage.SemanticIR
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ir); err != nil {
		return err
	}
	if ir.Schema != lineage.IRSchema || ir.SourcePath == "" || len(ir.Nodes) != 12 {
		return errors.New("INVALID_SEMANTIC_IR")
	}
	return nil
}

func writeReport(path string, report lineage.Report) error {
	raw, err := report.JSON()
	if err != nil {
		return err
	}
	return writeFile(path, append(raw, '\n'))
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func expectedDecision(caseName string) (string, bool) {
	decisions := map[string]string{
		"normal":                 lineage.StateClosed,
		"parent-missing":         lineage.StateUnknown,
		"stale-parent":           lineage.StateUnknown,
		"child-self-attestation": lineage.StateRefuted,
		"denominator-shrink":     lineage.StateRefuted,
		"unknown-upper-decision": lineage.StateUnknown,
		"explicit-contradiction": lineage.StateRefuted,
	}
	decision, ok := decisions[caseName]
	return decision, ok
}

func usage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: gooo-lineage <compile|evaluate|conformance|version>")
}
