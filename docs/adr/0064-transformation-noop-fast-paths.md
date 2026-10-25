# ADR-0064: Avoid buffers for unchanged transformation inputs

- **Status:** proposed
- **Date:** 2026-10-25 (expected; update on merge)
- **Version:** unreleased (post-v3.8.1)
- **PR:** Not opened
- **Issue(s):** No linked issue
- **Deciders:** Repository owner requested performance improvements without behavior or API changes; review pending
- **Category:** Perf

## Context and Problem

`urlEncode` allocates a builder even when every byte is already in its unchanged
set. `replaceComments` copies the input and its output even when there is no
opening block-comment delimiter. These transformations can run repeatedly while
inspecting request variables.

The owner also requested modernization using Go 1.27.3 without breaking changes.
Raising the module minimum would exclude Go 1.26 users and the existing TinyGo
configuration, so toolchain selection and the language minimum remain separate.

## Decision Drivers

- Preserve output bytes, the changed flag, errors, and public interfaces.
- Avoid allocating for input that requires no transformation.
- Preserve independent output ownership when the result changes.
- Measure both unchanged and changed inputs, including dense comments and escapes.
- Preserve minimum Go 1.26 and existing TinyGo compatibility.

## Considered Options

- Keep the existing unconditional buffers: simple but allocates on unchanged input.
- Locate the first changed byte or delimiter before allocating: chosen.
- Return a string backed by the entire temporary comment buffer: rejected because
  a short result could retain a large original allocation.
- Pool transformation output buffers: rejected because returned strings can outlive
  an invocation; reusing their backing storage could change previous results.
- Raise the minimum Go version to 1.27.3: rejected to preserve downstream compatibility.

## Decision Outcome

`urlEncode` scans the same byte allowlist as the existing encoder and returns the
input if all bytes are unchanged. Otherwise it copies the unchanged prefix into a
builder and encodes the suffix using the existing lowercase hex alphabet and
space-to-plus mapping. The capacity estimate applies threefold expansion only to
the suffix.

`replaceComments` locates the first `/*` delimiter with `strings.Index`. Without
one, it returns the original string and `changed == false`. Otherwise the existing
state machine starts just after that opener, with the output cursor at its original
position. Output copying is retained to avoid retaining a large input buffer for a
small result. No new `unsafe` use, pool, public API, or mutable shared state is added.

The workspace and modules prefer `toolchain go1.27.3`, retaining `go 1.26.0`.
Primary standard-Go CI uses 1.27.3; compatibility jobs still exercise 1.26.
`GOTOOLCHAIN=local` in compatibility workflows prevents the preferred toolchain from
silently upgrading those jobs. TinyGo remains on its existing Go 1.26 setup.
Reviewed `go fix` modernization uses typed atomic operations in a multipart test,
`slices.Backward` for an existing reverse traversal, and `strings.Builder` for
request formatting in the engine harness. These preserve traversal and output semantics.

## Measurements

Apple M3 Pro, darwin/arm64, Go 1.27.3, GOMAXPROCS=1. Medians of six
200 ms samples per case, comparing the pre-change and final implementations in
the same test binary. Both versions use the same inputs and benchmark loop.
They describe individual transformations, not complete WAF transactions.

| Case | Before ns/op | After ns/op | Before B/op | After B/op | Before allocs/op | After allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| URL/unchanged | 711.85 | 238.40 | 1024 | 0 | 1 | 0 |
| URL/late-space | 743.30 | 285.40 | 1024 | 352 | 1 | 1 |
| URL/short-space | 35.84 | 30.54 | 32 | 24 | 1 | 1 |
| URL/escaped | 272.80 | 274.95 | 384 | 384 | 1 | 1 |
| URL/mixed | 209.20 | 213.15 | 288 | 288 | 1 | 1 |
| Comments/unchanged | 355.95 | 8.37 | 704 | 0 | 2 | 0 |
| Comments/short | 22.68 | 21.79 | 3 | 3 | 1 | 1 |
| Comments/late | 377.55 | 92.69 | 768 | 768 | 2 | 2 |
| Comments/dense | 375.55 | 380.00 | 512 | 512 | 2 | 2 |
| Comments/unterminated | 177.50 | 174.30 | 248 | 248 | 2 | 2 |

The stable claim is zero allocations on unchanged inputs and reduced buffer
capacity for URL inputs with an unchanged prefix. Timing is workload- and
scheduler-dependent: immediately changed and comment-dense inputs do not receive
the same speedup. These measurements are not an end-to-end latency claim.
Raw paired measurements and the temporary comparison source are retained locally
in ignored `build/performance-go1273/`.

Permanent benchmark command:

```sh
GOMAXPROCS=1 go test ./internal/transformations -run '^$' \
  -bench '^(BenchmarkURLEncode|BenchmarkReplaceComments)$' \
  -benchmem -count=6 -benchtime=200ms
```

The existing URL benchmark's repeated outer `b.N` subbenchmark loop was replaced
with one subbenchmark per input and `b.Loop`, identically for the baseline and final
measurements. The added comment benchmark measures the allocation claim directly.

## Validation

Existing transformation table tests and SecLang fixtures remain the behavioral
home. A temporary differential check compares output, changed flag, and error
presence against the pre-change implementation for every one-byte and two-byte
input, all strings of length zero through six over `/*abc `, and 100,000 seeded
random inputs up to 1,024 bytes. The temporary check is not a second permanent test
suite. Benchmark cases cover unchanged input, short and late comments, dense
comments, unterminated comments, ASCII punctuation, UTF-8, and late URL changes.

Final local validation:

- `go run mage.go check`: passed, including the default/no-memoize suites, CRS
  tag matrix, example race tests, and lint with zero issues.
- `go test -race ./...` and a final `go test -race ./testing`: passed.
- `go vet ./...` and `go fix -diff ./...`: passed; no remaining fix suggestions.
- `GOTOOLCHAIN=go1.26.0 go test ./...`: passed on the retained minimum toolchain.
- Engine profiles and relevant core/parser tests under multiphase evaluation, plus
  engine profiles under no-regex-multiline: passed.
- Differential validation: 221,780 inputs per transformation passed on the final
  implementation, including output bytes and the changed flag.
- `go run mage.go adr`: passed.
- `betteralign` inspection/apply/recheck completed. Unrelated struct reorderings
  were reverted to their pre-tool snapshots; existing root-module layout
  suggestions remain intentionally unresolved to preserve layout contracts and
  avoid unmeasured changes. Example and CRS modules had no final diagnostics.
- Real TinyGo and the scanner Docker image were not run locally; TinyGo is not
  installed. Their configurations retain Go 1.26 compatibility or select the
  requested standard Go 1.27.3 image respectively.

## Technical Discussion

No substantive technical discussion recorded in a PR; this is a local change
requested by the repository owner.

## Consequences

Unchanged inputs can return without allocation. Changed outputs still require
owned storage; this does not make full request evaluation zero-allocation.
Scanning adds a small fixed cost for inputs that change immediately, so changed
cases are measured alongside the fast paths. Public struct layouts are preserved.

## References

- ADR-0019: existing zero-copy helpers and ownership context.
- ADR-0061: transformation caching by value identity.
- ADR-0063: completed RBL result-channel pooling, preserved unchanged.
- `internal/transformations/url_encode.go`
- `internal/transformations/replace_comments.go`
