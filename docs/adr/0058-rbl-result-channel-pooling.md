# ADR-0058: Pool completed RBL result channels

- **Status:** proposed
- **Date:** 2026-09-24 (expected; update on merge)
- **Version:** unreleased (post-v3.7.0)
- **PR:** Not opened
- **Issue(s):** No linked issue
- **Deciders:** Repository owner requested object pooling; review pending
- **Category:** Perf

## Context and Problem

Each RBL evaluation creates a buffered result channel to keep the DNS worker from
blocking if the caller times out. The repository owner requested object pooling
where it can reduce allocations without changing the established lookup behavior.
Transactions already have their own WAF-owned pool.

## Decision Drivers

- Reduce allocation on repeated evaluations of one compiled RBL operator.
- Preserve the 500 ms timeout, DNS query order, match result, TXT message and capture.
- Keep late DNS workers isolated from completed transactions and subsequent calls.
- Use the existing internal/sync abstraction without adding a global cache.

## Considered Options

- Keep allocating every result channel: simplest, but retains two avoidable allocations.
- Pool every channel in a deferred cleanup: unsafe because a timed-out worker may
  still send a result into a channel already used by a later evaluation.
- Reuse only channels whose sole result has been received: chosen.

## Decision Outcome

Each RBL operator owns an internal/sync.Pool of channels buffered for one result.
The worker sends exactly once and never closes or accesses the channel after that
send. Receiving the result is the ownership handoff: the caller can return the
empty channel to its pool without waiting for the worker's remaining stack unwind.
The received result is copied, and the worker never touches transaction state.

On timeout, the channel is not returned to the pool. The existing context
cancellation lets DNS work terminate; its one late send fits in the channel buffer.
The channel and any result then become collectible. There is no background draining
goroutine, wait on the timeout path, or cross-call result delivery.

A pool miss allocates a channel. Correctness never depends on sync.Pool retaining
objects across garbage collections. The pool is scoped to the operator, so unused
rule sets do not leave entries in a new process-global cache.

## Measurements

Apple M3 Pro, darwin/arm64, Go 1.27.1. Six samples before and after, using the same
benchmark fixture and leak-fixed implementation as the baseline. These are medians.

| Case | Before ns/op | After ns/op | Before B/op | After B/op | Before allocs/op | After allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| LookupTXT (local mock DNS) | 498343 | 438321.5 | 21974 | 21862.5 | 292 | 290 |
| InvalidHostname (no DNS I/O) | 5605 | 3966.5 | 824 | 681 | 16 | 14 |

The allocation claim is two fewer allocations per completed lookup in these
benchmarks. Timing is scheduler- and DNS-I/O-sensitive; no general latency or
end-to-end zero-allocation claim is made. Local DNS also allocates in the fixture,
so its memory figures are not a pure measurement of production client costs.
Timeout channels are deliberately not pooled and do not receive this saving.

```sh
go test ./internal/operators -run '^$' -bench '^BenchmarkRBL$' -benchmem -count=6
```

## Technical Discussion

No substantive technical discussion recorded in a PR; this is a local change
requested by the repository owner and supported by the benchmark measurements above.

## Participants

- Repository owner — requested safe object pooling.

## Consequences

- Successful and failed lookups that complete before timeout can reuse channels.
- Timed-out channels remain single-use; memory is not retained for their reuse.
- At most the active/concurrently cached channels are retained subject to sync.Pool
  garbage collection; the pool is not a hard concurrency or memory limit.
- Existing RBL behavior tests cover match results and capture. A concurrent mixed
  lookup case checks result isolation. The timeout case runs three successive
  virtual-time evaluations to detect stale-channel reuse and blocked workers.
- The tests require real Go; the RBL operator and its tests are excluded for TinyGo.

## Validation

- `go run mage.go check`, `go test -race ./...`, `go vet ./...` and
  `go run mage.go adr` passed.
- `TestRbl` passed ten repeated runs under the race detector, including concurrent
  positive/negative lookups, captures and successive timeouts.
- A temporary mutation that returned timed-out channels to the pool was detected
  by repeated timeout testing: a stale result returned at 0 ms instead of 500 ms.
- betteralign ran before/after its apply pass. Its layout rewrites were reverted;
  existing diagnostics and the RBL pointer-scan ordering suggestion are left
  unchanged because layout optimization is not needed for channel reuse.

## References

- ADR-0043: ownership and cleanup of cached resources.
- internal/operators/rbl.go
- internal/operators/rbl_test.go
- internal/sync/pool_std.go
