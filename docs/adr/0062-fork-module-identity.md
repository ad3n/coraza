# ADR-0062: Fork module identity and JSON dependency

- **Status:** proposed
- **Date:** 2026-09-24 (expected; update on merge)
- **Version:** unreleased (post-v3.7.0)
- **PR:** Not opened
- **Issue(s):** No linked issue
- **Deciders:** Repository owner requested this migration; review pending
- **Category:** Refactor

## Context and Problem

The repository owner requested the module path `github.com/ad3n/coraza/v3`
and the JSON dependency `github.com/ad3n/gjson`, retaining the v3 suffix.

## Decision Drivers

- Publish the fork under its own module identity.
- Preserve the existing WAF semantics and public package names.
- Make nested modules resolve the local checkout during verification.

## Considered Options

Only the explicitly requested module and dependency migration was considered.

## Decision Outcome

Migrate active imports and module declarations to `github.com/ad3n/coraza/v3`.
Use `github.com/ad3n/gjson v1.0.0` for JSON traversal and validation. Keep
historical ADRs and upstream attribution links unchanged. Use local replace
entries for examples and CRS tests because upstream version tags are not
available under the fork path.

The dependency requires Go 1.26. Update module and workspace minimums and CI
accordingly, including TinyGo 0.41.1. Update golangci-lint to v2.14.0 because
v2.6.2 cannot read the export data produced by the local Go 1.27 toolchain.

## Technical Discussion

No substantive technical discussion recorded in a PR; this is a local change
requested by the repository owner.

## Participants

- Repository owner — requested module and dependency migration.

## Consequences

- Consumers must migrate imports; old and new module types have distinct identities.
- Go 1.25 is no longer supported by the selected dependency.
- SecLang semantics, JSON error handling, and transaction ownership stay unchanged.
- Zero allocation is not promised by changing the JSON dependency.
- Publishing version tags for the fork remains release work.

## References

- ADR-0030: JSON recursion limits.
- [TinyGo Go compatibility matrix](https://tinygo.org/docs/reference/go-compat-matrix/).
- [Performance scan](../performance-scan.md).
