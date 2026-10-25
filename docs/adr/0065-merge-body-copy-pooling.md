# ADR-0065: Integrate pooled body copying while preserving local compatibility

- **Status:** proposed
- **Date:** 2026-10-25 (expected; update on merge)
- **Version:** unreleased (post-v3.8.1)
- **PR:** Not opened
- **Issue(s):** No linked issue
- **Deciders:** Repository owner requested the remote merge with local improvements and ADRs preserved; review pending
- **Category:** Perf

## Context and Problem

Origin commits c06dc52 and 2a8cb00 introduce reusable body-copy state, multipart
copy buffers, and transaction cleanup. The remote branch also removes the ADR
history and raises the minimum Go version. The local branch already includes
newer upstream security fixes, transformation fast paths, Go 1.27.3 toolchain
selection, and ADRs through 0064. Integrating the remote performance work must
preserve those decisions and the minimum supported Go version.

## Decision Drivers

- Preserve public interfaces, request limits, rule evaluation, and local fixes.
- Reuse temporary copy buffers without retaining request data across operations.
- Keep all existing ADRs and retain Go 1.26 compatibility.
- Measure allocation savings and disclose latency tradeoffs.

## Considered Options

- Accept the remote tree wholesale: rejected because it loses local fixes and ADRs.
- Keep all local implementations unchanged: avoids integration work but omits the
  remote copy-buffer allocation improvements.
- Integrate remote pooling and cleanup while resolving conflicts in favor of local
  compatibility and history: chosen.

## Decision Outcome

Each WAF owns a pool of limited-reader state and 32 KiB copy buffers. The copy
helper preserves byte limits, short-read EOF behavior, and reader errors. Each
borrowed state is returned after clearing its reader reference and buffer.
Multipart processing shares a processor whose only reusable mutable state is its
pool; each copy borrows its own writer wrapper and buffer and clears both before
returning them. Existing temporary-file registration and immediate close remain.
The TinyGo pool clears removed slots to avoid retaining borrowed objects.

Transaction close clears request-scoped references and transformation-cache
entries before returning the transaction to its pool. Saved match slices are not
mutated. Closing an unreused, already-closed transaction returns without a second
pool insertion; accessing a transaction after Close remains unsupported. File
buffer reset attempts file removal even if Close fails, joining cleanup errors.
Nil body readers now return errors where they previously could panic. These are
error-path robustness changes, not new rule semantics or public signatures.

The local no-op URL and comment fast paths, RBL pooling, module identity, newer
upstream fixes, CRS v4.30.0, and x/net v0.60.0 are retained. Remote gjson and
libinjection fork updates are integrated. The remote jsonschema replacement
requires Go 1.27 and is excluded: kaptinlin/jsonschema v0.4.6 remains. All modules
retain go 1.26.0 and prefer toolchain go1.27.3. Existing ADR files are preserved;
ignore rules no longer hide them. JavaScript octal decoding accumulates bytes
without a temporary numeric parse, preserving the existing digit limit.

## Measurements

Apple M3 Pro, darwin/arm64, Go 1.27.3, GOMAXPROCS=1. Median of three 200 ms
samples from the remote's paired BenchmarkBodyCopy (false uses io.CopyN; true
uses the merged helper), after concurrent race validation completed:

| Body | Before ns/op | Pooled ns/op | Before B/op | Pooled B/op | Before allocs/op | Pooled allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 KiB | 243.2 | 835.3 | 1048 | 0 | 2 | 0 |
| 64 KiB | 5960 | 2169 | 32794 | 1 | 2 | 0 |

The helper reduces allocations, but clearing a full reusable buffer makes the
small-body case slower. This is not a universal throughput improvement or a claim
that whole transactions allocate nothing. Pool allocation amortization and
runtime eviction can produce nonzero bytes per operation.

```sh
GOMAXPROCS=1 go test ./internal/corazawaf -run '^$' \
  -bench '^BenchmarkBodyCopy/(1024|65536)/(false|true)$' \
  -benchmem -count=3 -benchtime=200ms
```

## Validation

- Full mage check passes: lint, default/no-memoize suites, CRS tag matrix,
  example race tests, and targeted operator tags.
- Root race tests, go vet, and all root tests with Go 1.26 pass.
- Body processor, core, and engine tests with no_fs_access pass.
- Standard-Go tests selecting the tinygo pool implementation pass. A real TinyGo
  toolchain is not installed, so this does not validate TinyGo compilation.
- The libinjection fork's own tests pass. An external differential program compares
  upstream v0.3.4 with fork v1.0.0 on 496 corpus inputs plus 100,000 seeded random
  inputs: SQLi detection, fingerprints, and XSS detection are identical.
- Existing behavioral test homes cover multipart isolation, body-copy limits and
  errors, cleanup, and transformation outputs. Remote regression cases are kept.
- Betteralign inspection/apply/recheck completed; unrelated layout changes were
  restored to avoid changing public layouts or expanding merge scope.

## Technical Discussion

No substantive technical discussion recorded in a PR; this integration was
requested locally by the repository owner.

## Consequences

Temporary body copying can reuse storage with zero measured allocations after
pool warm-up. Clearing buffers limits stale data retention but has a measurable
cost for small bodies. Pools may retain 32 KiB per cached state until eviction.
Maintaining Go 1.26 compatibility postpones the remote jsonschema migration.
No existing ADR is renumbered, replaced, or deleted by this merge.

## References

- Remote performance commits: c06dc52 and 2a8cb00.
- ADR-0062: fork module identity.
- ADR-0063: RBL result-channel pooling.
- ADR-0064: transformation no-op fast paths and Go toolchain compatibility.
- internal/corazawaf/body_copy.go
- internal/bodyprocessors/multipart.go
