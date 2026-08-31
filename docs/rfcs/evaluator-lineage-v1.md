# Evaluator lineage v1

## Problem

Self-improvement must not allow a new evaluator to be the authority that
approves itself. The parent evaluator is the upper authority, and its identity
must remain reproducible after the child candidate changes.

## Input boundary

`gooo/evaluator-lineage/input/v1` accepts a parent reference and a child
candidate reference. Each reference contains an ID, `vX.Y.Z` release tag,
release digest, observed release digest, generation depth, and inherited
counterexample count. Only the tag/digest pair identifies a release. A parent
reference is stale when its observed digest differs from its pinned digest.

The evaluator has no branch, pull-request, checkout, or mutable-ref field.
Missing or stale parent evidence is `UNKNOWN`, not an approval. A child ID or
digest collision with its parent and a child self-attestation are `REFUTED`.

## Denominator

The denominator is immutable at twelve cells. FOUNDATION, COHERENCE, and
REGRESSION each contain four cells. DRIVER, OUTCOME, and GUARDRAIL each
contain four cells. Every cell has a `.gooo` activity, metric ID, metric path,
artifact, and evaluator binding. No metric is created outside the semantic
source.

## Resolution

Cell decisions are exact categorical observations: `ACCEPTED`, `REJECTED`, or
`UNKNOWN`. A rejected cell is an explicit contradiction. Missing, stale, or
unknown upper authority produces a six-field UNKNOWN record. The final order
is `REFUTED > UNKNOWN > CLOSED`, so a contradiction is never hidden by a
different accepted cell.

Improvement is not inferred. It is `UNKNOWN` without exact before and after
values, their digests, an input digest, and a metric bound to the denominator.
The evaluator does not emit scores or percentages.

## Provenance

The report records the digest chain from `.gooo` source to semantic IR,
generated Go, evaluator, and human report. The runtime reports zero repository
writes, zero local test executions, and zero cross-project required gates.
Inventory checks exclude only the absence of a project-root README.
