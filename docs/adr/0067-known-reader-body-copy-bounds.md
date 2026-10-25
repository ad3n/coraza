# ADR-0067: Bound pooled copying by known remaining reader length

- **Status:** proposed
- **Date:** 2026-10-25 (expected; update on merge)
- **Version:** unreleased (post-v3.8.1)
- **PR:** Not opened
- **Issue(s):** No linked issue
- **Deciders:** Repository owner requested small-body optimization with large copy limits and unchanged API and logic; review pending
- **Category:** Perf

## Context and Problem

ADR-0066 limits temporary buffer exposure and scrubbing by the requested copy
limit. A small body with a large configured limit still exposes and clears 32 KiB.
Common memory-backed readers already know exactly how many bytes remain, so that
information can reduce temporary storage exposure without probing a stream.

## Decision Drivers

- Preserve public interfaces, copied bytes, count, EOF, and error behavior.
- Preserve zero measured allocations for warmed memory-backed copies.
- Clear the entire buffer region exposed to readers, not only returned bytes.
- Avoid new read calls, custom copy loops, and assumptions about arbitrary readers.

## Considered Options

- Adaptive small first reads for every reader: rejected here because they change
  stream read/write chunk boundaries and can change partial progress when an I/O
  error occurs. The request requires unchanged logic.
- Clear only the reported byte count: rejected because io.Reader may use its
  entire supplied slice as scratch, including beyond the returned count.
- Trust any Len method: rejected because an arbitrary implementation can have
  different semantics or side effects.
- Use remaining lengths of three concrete standard-library reader types: chosen.

## Decision Outcome

For *bytes.Reader, *bytes.Buffer, and *strings.Reader, the helper reads Len before
borrowing its copy state. Its temporary buffer length and capacity become
min(limit, max(1, remaining), 32 KiB). All other reader types retain the ADR-0066
bound. This is deliberately a concrete-type switch, not a new public interface.

The LimitedReader retains the original requested limit. A short body therefore
still returns its byte count and io.EOF, rather than incorrectly reporting a
successful full-limit copy. The one-byte minimum permits io.CopyBuffer to observe
EOF for empty readers without rejecting a zero-length buffer. Nonpositive limits
and nil readers keep their existing paths. The same defer clears the entire
exposed buffer and drops the reader reference before pool return.

These concrete readers return min(remaining, supplied length), so reducing the
buffer to remaining bytes preserves the chunks delivered to BodyBuffer.Write.
For larger readers, the 32 KiB cap is unchanged. There is no speculative Read,
WriterTo bypass, destination shortcut, or modification of reader position during
length inspection. Calls do not introduce mutable shared state.

Opaque wrappers, custom Len implementations, network streams, and other readers
whose length is unknown keep the general path. This change does not claim to
accelerate every short stream. Preserving their existing chunk/error behavior and
scrubbing the full exposed region takes precedence over assuming a length.

## Measurements

Apple M3 Pro, darwin/arm64, Go 1.27.3, GOMAXPROCS=1. Medians of five 200 ms
samples, using the same expanded BenchmarkBodyCopy source before and after the
implementation change. Source: bytes.Reader. Copy limit: 1 MiB for every row.
io.CopyN figures are from the after run.

| Body bytes | io.CopyN ns/op | Previous pool ns/op | Known-length pool ns/op | Before pool allocs/op | After pool allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: |
| 64 | 3718 | 178.9 | 29.41 | 0 | 0 |
| 1024 | 3969 | 203.8 | 62.34 | 0 | 0 |
| 4096 | 4044 | 349.6 | 165.6 | 0 | 0 |
| 65536 | 6148 | 1970 | 2014 | 0 | 0 |

Small cases report 0 B/op; 64 KiB reports 1 B/op amortized with rounded zero
allocs/op. The 1 KiB case improves about 3.3 times over the previous pooled helper.
The large case retains its 32 KiB buffer and shows about 2% timing variation;
there is no large-body speedup claim. io.CopyN reports two allocations per copy.

```sh
GOMAXPROCS=1 go test ./internal/corazawaf -run '^$' \
  -bench '^BenchmarkBodyCopy/(64|1024|4096|65536)/limit=1048576/(false|true)$' \
  -benchmem -count=5 -benchtime=200ms
```

This is a helper microbenchmark with a reusable destination. First use, pool
misses or GC eviction, buffer growth, and filesystem operations may allocate.
Raw paired logs are retained locally under ignored build/performance-large-limit/.

## Validation

The existing TestBodyReaderWriteFromReader table gains large-limit rows for each
supported reader, an empty buffer, and an opaque reader fallback. It asserts bytes,
EOF, consumed input, and complete pooled-state cleanup. Existing exact/short/limited
reads, nonpositive limits, nil readers, reader errors, and file cases remain.
No duplicate test home or new test function is introduced.

Final local validation:

- `go run mage.go check`: passed, including lint with zero issues, default and
  no-memoize tests, the CRS tag matrix, and example race tests.
- `go test -race ./...` and `go vet ./...`: passed.
- `GOTOOLCHAIN=go1.26.0 go test ./...`: passed.
- Core and engine tests with `no_fs_access`: passed.
- The copy table with standard Go's `tinygo` build tag: passed. Real TinyGo
  compilation remains unverified because the toolchain is not installed.
- Existing parallel copy benchmarks at 64 B, 1 KiB, and 64 KiB passed under the
  race detector with GOMAXPROCS=4, including output-isolation assertions.
- `go run mage.go adr`: passed for all 67 records.
- Betteralign inspection/apply/recheck completed. Unrelated layout rewrites were
  restored to preserve public layout contracts and scope; pre-existing root
  diagnostics remain. Example and CRS modules had no final diagnostics.

## Technical Discussion

No substantive technical discussion recorded in a PR; this follows the repository
owner's request to optimize large-limit copies without changing logic or APIs.

## Consequences

Memory-backed small bodies avoid exposing and clearing the unused 32 KiB tail,
even with a large configured copy limit. Unknown readers retain existing behavior
and costs. The type switch adds a small dispatch cost; pool size and ownership
remain unchanged. ADR-0065 and ADR-0066 remain historical records of the prior
measurements and complementary pooling decisions.

## References

- ADR-0065: pooled body-copy ownership and cleanup.
- ADR-0066: buffer scrubbing bounded by copy limit.
- internal/corazawaf/body_copy.go
- internal/corazawaf/body_buffer_test.go
