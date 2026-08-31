# Gooo evaluator lineage

`gooo-evaluator-lineage` is a public, self-contained Go 1.27 evaluator for
self-improvement lineage. An immutable parent evaluator judges a child
evaluator candidate. A candidate cannot approve itself: the parent release
must be a distinct, pinned release tag and `sha256:` digest.

The only cross-repository identity accepted by the evaluator is an immutable
release tag paired with its digest. Branch names, pull requests, working-tree
state, and another repository's branch status are not inputs or required
gates. This repository therefore has `cross_project_required_gates=0`.

## Authority chain

The checked-in chain is explicit and digest-bound:

```text
examples/evaluator-lineage/main.gooo
  -> internal/generated/semantic-ir.json
  -> internal/generated/semantic.gooo.go
  -> internal/lineage/evaluate.go
  -> artifacts/**/report.md
```

`compile` validates the `.gooo` source against the fixed denominator, emits
semantic IR, and emits generated Go. The evaluator verifies the source,
semantic IR, generated Go, contract, and evaluator digests before judging the
lineage input. Every denominator metric is bound to a `.gooo` activity;
unbound metrics are rejected.

## Fixed 12-cell denominator

There are exactly twelve cells. Each proof choice and each indicator class
has exactly four cells. The deliberately rectangular distribution is part of
the contract: FOUNDATION/COHERENCE/REGRESSION are balanced, and
DRIVER/OUTCOME/GUARDRAIL are balanced.

| proof choice | cells | indicator distribution |
|---|---:|---|
| FOUNDATION | 4 | DRIVER 2, OUTCOME 1, GUARDRAIL 1 |
| COHERENCE | 4 | DRIVER 1, OUTCOME 2, GUARDRAIL 1 |
| REGRESSION | 4 | DRIVER 1, OUTCOME 1, GUARDRAIL 2 |
| Total | 12 | DRIVER 4, OUTCOME 4, GUARDRAIL 4 |

No score, percentage, or inference-based improvement is computed. An
improvement is `UNKNOWN` unless the input includes an exact before/after pair
with immutable digests and a denominator-bound metric.

## Decision rules

The precedence is fixed: `REFUTED > UNKNOWN > CLOSED`.

| fixture | expected decision |
|---|---|
| normal | CLOSED |
| parent-missing | UNKNOWN |
| stale-parent | UNKNOWN |
| child-self-attestation | REFUTED |
| denominator-shrink | REFUTED |
| unknown-upper-decision | UNKNOWN |
| explicit-contradiction | REFUTED |

Every `UNKNOWN` record preserves `stage`, `step`, `reason`, `unknown_class`,
`next_operation`, and `blocked_by`. Human reports include parent/child
digests, generation depth, inherited counterexample count, exact accepted,
rejected, and unknown counts, plus the read-only runtime boundary:
`repository_writes=0`, `local_test_executions=0`, and
`cross_project_required_gates=0`.

The inventory contract explicitly excludes a missing project-root README from
inventory violations (`PROJECT_ROOT_README`); other source and binding gaps
remain actionable.

## Release integrity

The first release, `v0.1.0`, is preserved but its GitHub API state is
`immutable=false`; it is therefore `REFUTED_RELEASE_IMMUTABILITY` and must not
be treated as a successful immutable release. The audit and next-release
policy are recorded in
`provenance/release-history-provenance-v1.json`. The repository immutable
releases setting is enabled through the official API, and only a later release
that is `immutable=true` in both REST and GraphQL can satisfy the release
gate.

## Verification

All development verification runs in GitHub Actions with Go 1.27:

```text
go test ./...
gofmt -d ...
go vet ./...
gooo-lineage compile
gooo-lineage conformance
```

Local Go build, test, formatting, vet, and conformance executions are not part
of the authoring workflow. The CI summary and uploaded reports are the
verification authority.
