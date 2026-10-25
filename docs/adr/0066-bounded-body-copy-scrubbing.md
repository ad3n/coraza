# ADR-0066: Bound pooled body-copy buffer scrubbing by the copy limit

- **Status:** proposed
- **Date:** 2026-10-25 (expected; update on merge)
- **Version:** unreleased (post-v3.8.1)
- **PR:** Not opened
- **Issue(s):** No linked issue
- **Deciders:** Repository owner requested faster small-body pooling without API or logic changes; review pending
- **Category:** Perf

## Context and Problem

ADR-0065 records that pooling body-copy state eliminates steady-state allocations
but clears 32 KiB even when the requested copy limit is only 1 KiB. That fixed
cleanup cost makes small copies slower than io.CopyN. The pool and its ownership
model are retained; this decision refines the buffer exposure and cleanup bound.

## Decision Drivers

- Keep copy counts, byte limits, error behavior, and public interfaces unchanged.
- Preserve zero measured allocations for warmed, memory-backed copies.
- Clear every byte a reader could have written before returning state to the pool.
- Avoid a second pool, new allocation, custom copy loop, or unsafe code.

## Considered Options

- Keep clearing all 32 KiB: correct but wasteful for small copy limits.
- Clear only the reported written count: rejected because a reader can modify its
  supplied buffer beyond the returned count, including when it returns an error.
- Separate small-buffer pools: adds thresholds and retention without being needed.
- Bound both buffer length and capacity by min(limit, 32 KiB), then clear that
  entire exposed slice: chosen.

## Decision Outcome

For a positive limit the helper borrows the same pooled state and creates a full
slice expression with length and capacity equal to min(limit, 32 KiB). io.CopyBuffer
continues using the existing LimitedReader and BodyBuffer.Write. LimitedReader
already limits each Read to the remaining byte count, so the readable slice length
and copy chunk boundaries remain unchanged. The bounded capacity prevents a reader
from reslicing into the unexposed tail. Deferred cleanup clears the full exposed
slice and removes the reader reference, including on errors or panics.

New pool states are zero initialized. By induction, every previously exposed byte
is cleared before return, and the unexposed tail stays zero. Alternating large and
small copies therefore leaves no body data behind without clearing unused bytes
again. Nonpositive limits return (0, nil) without acquiring a pool state, matching
the previous behavior without calling the reader or writer. Positive nil-reader
limits keep their existing error. No public API or rule logic changes.

The bound is the requested copy limit, not the observed body length. A short or
empty reader with a large limit still incurs full-buffer cleanup. First use, GC
pool eviction, destination growth, and file-backed copies may allocate; the
zero-allocation claim is scoped to the warmed helper benchmark.

## Measurements

Apple M3 Pro, darwin/arm64, Go 1.27.3, GOMAXPROCS=1. Medians of five 200 ms
samples per case. The existing paired BenchmarkBodyCopy was expanded with small
sizes and the 32 KiB boundary; identical benchmark source was run before and after
the implementation change. io.CopyN numbers below come from the after run.

| Bytes | io.CopyN ns/op | Previous pool ns/op | Bounded pool ns/op | io.CopyN allocs/op | Bounded pool allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: |
| 64 | 53.13 | 772.2 | 27.17 | 2 | 0 |
| 1024 | 232.3 | 798.8 | 60.82 | 2 | 0 |
| 4096 | 654.2 | 947.5 | 166.5 | 2 | 0 |
| 32768 | 4800 | 1399 | 1316 | 2 | 0 |
| 65536 | 5862 | 2269 | 1942 | 2 | 0 |

The bounded pool reports 0 B/op through 32 KiB and 1 B/op amortized at 64 KiB;
allocs/op rounds to zero in all cases. Small copies improve by removing unnecessary
scrubbing; changes at or above 32 KiB are measurement variation, not an algorithmic
speedup claim. These are helper microbenchmarks, not end-to-end WAF throughput.

```sh
GOMAXPROCS=1 go test ./internal/corazawaf -run '^$' \
  -bench '^BenchmarkBodyCopy/[0-9]+/(false|true)$' \
  -benchmem -count=5 -benchtime=200ms
```

## Validation

The existing TestBodyReaderWriteFromReader table is the behavioral home. It covers
exact and short reads, limited reads, nil and nonpositive limits, reader errors,
and file-backed copies. A multi-buffer row precedes small rows, and the test
retains references to allocated pool states to assert that all reader references
and all buffer bytes are cleared after each invocation. No duplicate test suite
or new test function is introduced. Existing parallel benchmarks check isolation.

Final local validation:

- `go run mage.go check`: passed, including lint with zero issues and the full
  default/no-memoize and CRS tag matrix.
- `go test -race ./...` and `go vet ./...`: passed.
- `GOTOOLCHAIN=go1.26.0 go test ./...`: passed.
- Core and engine suites with `no_fs_access`: passed.
- The existing copy table under standard Go with the `tinygo` tag: passed.
  Real TinyGo compilation remains unverified because TinyGo is not installed.
- Existing parallel copy benchmarks for 64 B, 1 KiB, and 64 KiB under the race
  detector with GOMAXPROCS=4: passed, including output-isolation assertions.
- `go run mage.go adr`: passed for all 66 records.
- Betteralign inspection, apply, and recheck completed. Unrelated layout rewrites
  were restored; existing root layout suggestions remain to preserve public
  layouts and task scope. Example and CRS modules report no final diagnostics.
- Raw before/after benchmark logs are retained locally in ignored
  `build/performance-small-body/`.

## Technical Discussion

No substantive technical discussion recorded in a PR; the repository owner
requested this optimization after reviewing the ADR-0065 benchmark tradeoff.

## Consequences

Small bounded copies retain zero measured allocations and avoid the 32 KiB
scrubbing penalty. Large-limit copies keep the same buffer size and cleanup cost.
Buffer capacity available to a custom reader is now bounded; the io.Reader
contract, supplied readable length, copy results, and error handling are preserved.
Pool storage remains 32 KiB per cached state, with no new retention class.

## References

- ADR-0065: pooled body copying and historical small-body measurements.
- internal/corazawaf/body_copy.go
- internal/corazawaf/body_buffer_test.go
