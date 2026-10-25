# Evaluation of 32 KiB pooled-buffer scrubbing

Date: 2026-10-25. Production baseline: `55936aa`.

**Decision: retain the existing full `clear(buffer)`. No additional production
optimization was justified by these measurements.** The requested small-stream
cleanup improvement remains unproven under unchanged I/O and cleanup constraints.
The owner chose to preserve the local deletion of `docs/adr/`; this report records
the evaluation without restoring or rewriting those records.

## Constraints

An unknown-length reader can use every byte of the slice supplied to Read as
scratch, including bytes beyond its returned count or on an error. Clearing only
reported bytes would weaken the existing pool cleanup guarantee. Smaller initial
reads would change I/O chunk boundaries and potentially partial progress on
errors. Neither approach was accepted. Deferred cleanup must leave all exposed
bytes zero before returning state to the pool, without retaining reader references.

## Experiments

Apple M3 Pro, darwin/arm64, Go 1.27.3, GOMAXPROCS=1. Temporary microbenchmarks were
run in the core package, then removed from the normal test suite. Fixtures and
state allocation were outside the measured loop. Every strategy reported 0 B/op
and 0 allocs/op in the small cases.

Five samples of 100 ms per case. Each iteration fills a prefix of a 32 KiB buffer
with nonzero bytes and performs cleanup. These timings include the common fixture
fill loop; they are not isolated clear latency or end-to-end request timings.

| Nonzero prefix | Full clear ns/op | Copy immutable zeros ns/op | Check suffix after 1 KiB ns/op | Check suffix after 4 KiB ns/op | Check suffix after prefix ns/op |
| --- | ---: | ---: | ---: | ---: | ---: |
| 64 B | 195.6 | 583.3 | 442.3 | 429.1 | 461.8 |
| 1 KiB | 453.8 | 842.5 | 729.3 | 698.3 | 730.4 |

The suffix-check variants compare the complete suffix with an immutable zero
string. Only when that suffix is already zero do they clear the prefix alone;
otherwise they clear the whole exposed buffer. This avoids trusting the returned
byte count, but the inspection cost outweighs the saved stores in these samples.
Copying a zero block is also slower. Neither implementation was retained.

A separate layout experiment compared the existing state with a padded layout,
using a 1 KiB copy followed by a full clear. Twelve 200 ms samples produced:

| Layout | Median ns/op | Minimum ns/op | Maximum ns/op |
| --- | ---: | ---: | ---: |
| Existing | 783.30 | 720.7 | 818.6 |
| Padded | 776.05 | 705.2 | 822.3 |

The median difference is below 1% and sample ranges overlap substantially. This
experiment does not establish an improvement or guarantee heap/address alignment;
it does not justify retaining padding or architecture-specific assumptions. Its
absolute timings must not be compared directly with the different fixture loop
above. No claim of statistical significance or cross-platform performance is made.

The existing Go runtime already provides an architecture-specific implementation
of clear. These results reject the tested alternatives on this machine, not every
possible future optimization. A future change needs measured improvement plus the
same scratch-cleanup and I/O-contract checks.

## Validation

The production Go files are unchanged. The existing contract table checks partial
reads, data with EOF/error, write-error precedence, byte consumption, and complete
pool cleanup even when a reader writes scratch beyond the count it returns.

- `go test -race ./...`: passed.
- `go vet ./...`: passed.
- `GOTOOLCHAIN=go1.26.0 go test ./...`: passed.
- Core and engine suites with `no_fs_access`: passed.
- The copy contract table with standard Go's `tinygo` tag: passed. Real TinyGo is
  not installed and was not validated.
- The retained opaque stream-copy benchmark passed for 64 B, 1 KiB, and 64 KiB
  bodies with a 1 MiB limit: all reported 0 allocs/op. Small bodies reported
  0 B/op; 64 KiB reported 1 B/op amortized.
- ADR validation is not reported as passing: the owner explicitly chose to keep
  the local ADR deletions. Those unrelated deletions are left unstaged.

The production allocation check uses an opaque reader without Len or WriterTo:

```sh
GOMAXPROCS=1 go test ./internal/corazawaf \
  -run '^TestBodyReaderWriteFromReader$' \
  -bench '^BenchmarkBodyCopy/(64|1024|65536)/limit=1048576/stream=true/true$' \
  -benchmem -count=3 -benchtime=200ms
```

Zero measured allocations apply to a warmed pool and reusable memory destination;
pool misses, GC eviction, destination growth and filesystem I/O may allocate.
The temporary candidate source and raw logs are retained locally in ignored
`build/performance-scrubbing/`. No rejected implementation or duplicate permanent
test suite is shipped.
