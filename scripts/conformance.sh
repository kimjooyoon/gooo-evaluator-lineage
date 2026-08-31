#!/usr/bin/env bash
set -euo pipefail

jq -e '
  .schema == "gooo/evaluator-lineage/release-history-provenance/v1" and
  .audit.source == "gooo-self-improvement-ledger API audit" and
  .releases[0].tag == "v0.1.0" and
  .releases[0].immutable == false and
  .releases[0].decision == "REFUTED_RELEASE_IMMUTABILITY" and
  .next_release_policy.next_release_tag == "v0.1.1" and
  .next_release_policy.v0_1_0_evidence_counts_as_success == false
' provenance/release-history-provenance-v1.json >/dev/null

go test ./...

go_files=$(git ls-files '*.go')
gofmt_output=$(gofmt -d ${go_files} || true)
if [[ -n "${gofmt_output}" ]]; then
  printf '%s\n' "${gofmt_output}"
  exit 1
fi

go vet ./...
go run ./cmd/gooo-lineage compile \
  -source examples/evaluator-lineage/main.gooo \
  -contract contracts/evaluator-lineage-denominator-v1.json \
  -output-ir internal/generated/semantic-ir.json \
  -output-go internal/generated/semantic.gooo.go
git diff --exit-code -- internal/generated/semantic-ir.json internal/generated/semantic.gooo.go

go run ./cmd/gooo-lineage conformance \
  -fixtures fixtures \
  -output-dir artifacts/conformance

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  cat artifacts/conformance/ci-summary.md >> "${GITHUB_STEP_SUMMARY}"
fi
