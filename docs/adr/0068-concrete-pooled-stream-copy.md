# ADR-0068: Remove generic copy dispatch from pooled body streaming

- **Status:** proposed
- **Date:** 2026-10-25 (expected; update on merge)
- **Version:** unreleased (post-v3.8.1)
- **PR:** Not opened
- **Issue(s):** No linked issue
- **Deciders:** Repository owner requested optimization for unknown-length streams without logic or API changes; review pending
- **Category:** Perf

## Context and Problem

ADR-0067 optimizes readers whose remaining length is known. Opaque streams still
need the general bounded read path and full cleanup of their exposed scratch
buffer. The helper currently routes its concrete LimitedReader and BodyBuffer
through io.CopyBuffer, which performs generic interface dispatch despite neither
endpoint supporting its optional WriterTo/ReaderFrom shortcuts.

## Decision Drivers

- Preserve read sizes, byte consumption, write ordering, partial progress, and errors.
- Keep zero measured steady-state allocations for unknown-length memory-backed copies.
- Clear every exposed scratch byte before returning state to the pool.
- Avoid guessing stream length, changing buffer sizes, or adding public API.

## Considered Options

- Adaptive smaller initial reads: retain the ADR-0067 rejection because changing
  read/write chunk boundaries can affect partial progress on I/O errors.
- Clear only returned bytes: rejected because readers may use the full slice as
  scratch, including bytes beyond their reported count.
- Add padding to align the buffer: experimentally evaluated, but the small timing
  differences did not justify extra layout assumptions; not retained.
- Keep io.CopyBuffer: simplest, but retains unnecessary generic dispatch.
- Use the same bounded copy loop with concrete endpoints: chosen for a measured,
  modest reduction in dispatch overhead without changing I/O boundaries.

## Decision Outcome

The helper calls the existing io.LimitedReader.Read and BodyBuffer.Write directly.
The pool, buffer size, capacity, reader-length optimizations, and cleanup defer are
unchanged. A positive read is written before examining its accompanying error.
A write error takes precedence over a simultaneous read error, short writes produce
io.ErrShortWrite, and zero-byte nil-error reads continue just as before. EOF ends
the copy; the existing outer normalization still returns success when the requested
limit was satisfied and io.EOF for a short successful stream.

BodyBuffer is a concrete internal implementation, not a user-supplied writer. Its
Write returns valid counts through bytes.Buffer or os.File. Generic Copy's invalid
writer-count guard is therefore unnecessary here; custom reader behavior still
passes through the same LimitedReader, with the same read slice and call order.
No WriterTo shortcut is newly used, and BodyBuffer has no ReaderFrom implementation.
If these concrete endpoint contracts change in future, revisit this specialization.

All bytes of the exposed buffer are still cleared on return or panic and the
reader reference is removed before pool insertion. Unknown streams still receive
up to 32 KiB per read. There is no probe, additional read, deferred write, alternate
pool, unsafe operation, or public API change.

## Measurements

Apple M3 Pro, darwin/arm64, Go 1.27.3, GOMAXPROCS=1. The source is a bytes.Reader
hidden behind an io.Reader-only wrapper, so it has no visible Len or WriterTo.
Every copy has a 1 MiB limit and a reusable in-memory destination. Medians of eight
200 ms samples compare the exact previous helper and the new helper in one test
binary; wrapper conversion and setup occur outside the measured loop.

| Body bytes | Previous pooled ns/op | Concrete pooled ns/op | Before allocs/op | After allocs/op | After B/op |
| --- | ---: | ---: | ---: | ---: | ---: |
| 64 | 226.50 | 207.95 | 0 | 0 | 0 |
| 1024 | 238.95 | 228.40 | 0 | 0 | 0 |
| 65536 | 1998.0 | 1988.5 | 0 | 0 | 1 |

The measured reduction is about 8% at 64 B and 4% at 1 KiB; the large-body result
is effectively unchanged. Full-buffer scrubbing still dominates small unknown
streams, so this is not the much larger known-length improvement from ADR-0067.
Zero allocs/op is rounded steady-state measurement, not a guarantee for pool
misses, GC eviction, destination growth, or file-backed I/O. Small cases also
report 0 B/op; the large case amortizes about 1 B/op.

The existing benchmark now includes both known readers and opaque stream wrappers:

```sh
GOMAXPROCS=1 go test ./internal/corazawaf -run '^$' \
  -bench '^BenchmarkBodyCopy/(64|1024|65536)/limit=1048576/stream=true/(false|true)$' \
  -benchmem -count=5 -benchtime=200ms
```

The final true/false component selects pooled copying versus io.CopyN. Raw logs,
the discarded alignment experiment, and temporary same-binary comparison source
are retained locally in ignored build/performance-stream/. The historical helper
is not added as a second permanent implementation or test suite.

## Validation

The behavioral home remains TestBodyReaderWriteFromReader. Added rows cover opaque
one-byte reads, data returned with EOF, data returned with an error, a simultaneous
write/read failure, and a full-limit read accompanied by an error. A test reader
fills the entire exposed slice before returning fewer bytes to verify that cleanup
still removes scratch bytes beyond the reported count. Explicit read-size sequences
check that unknown streams retain their 32 KiB chunks and limited reads retain
their requested bound. Existing assertions cover counts, errors, output, reader
consumption, and complete pool-state cleanup.

These same table cases pass against the previous implementation and the new one.
The test reader adapter is local test support; no new test function or duplicate
behavioral home is introduced.

Final local validation:

- `go run mage.go check`: passed, including zero lint issues, default/no-memoize
  suites, CRS tag matrix, and example race tests.
- `go test -race ./...` and `go vet ./...`: passed.
- `GOTOOLCHAIN=go1.26.0 go test ./...`: passed.
- Core and engine tests with `no_fs_access`: passed.
- The copy table under standard Go with the `tinygo` tag: passed. Real TinyGo
  compilation remains unverified because the toolchain is not installed.
- Existing parallel copy benchmarks at 64 B, 1 KiB, and 64 KiB passed under the
  race detector with GOMAXPROCS=4, including output-isolation assertions.
- `go run mage.go adr`: passed for all 68 records.
- Betteralign inspection/apply/recheck completed. Unrelated layout rewrites were
  restored; pre-existing root diagnostics remain to preserve public layouts and
  scope. Example and CRS modules had no final diagnostics.
- Contract table cases passed separately against the pre-change helper and final
  helper; the eight-sample paired benchmark confirmed zero measured allocations.

## Technical Discussion

No substantive technical discussion recorded in a PR; this optimization follows
the repository owner's request after ADR-0067 preserved the unknown-stream path.

## Consequences

Unknown streams gain a modest reduction in copy overhead while keeping the same
allocation and cleanup guarantees. The explicit loop adds maintenance responsibility
for preserving the standard copy semantics used by these concrete endpoints.
Chunk-size changes and relaxed scrubbing are deliberately excluded. The previous
ADRs remain historical records and document complementary buffer-bound decisions.

## References

- ADR-0065: pooled body-copy ownership and cleanup.
- ADR-0066: copy-limit buffer bounds.
- ADR-0067: known-reader bounds and opaque-stream behavior.
- internal/corazawaf/body_copy.go
- internal/corazawaf/body_buffer_test.go
