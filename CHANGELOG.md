# Changelog

## Unreleased

### Features

* Generated URLs now include the path segment from `Ingress` rules,
  `HTTPRoute` matches (Exact / PathPrefix; regex matches skipped), and
  Traefik `IngressRoute` `Path()` / `PathPrefix()` matchers
  ([#33](https://github.com/home-operations/gatus-sidecar/issues/33)).
  Trivial paths (`/` and empty) are ignored so single-host probes are
  unchanged.
* Per-resource-type endpoint-name prefixes via `--prefix-ingress`,
  `--prefix-service`, `--prefix-httproute`, and `--prefix-ingressroute`.
  Disambiguates the case where, for example, an Ingress and a Service share
  a `metadata.name` and would otherwise collide in Gatus
  ([#59](https://github.com/home-operations/gatus-sidecar/issues/59)).
* `--gateway-name` and `--ingress-class` may be supplied multiple times.
  Each flag now acts as a set: a resource passes the filter if its gateway
  reference / ingress class matches *any* of the configured values.
* `--log-level=debug|info|warn|error` flag to control verbosity (default
  `info`). Logs are emitted as `log/slog` structured records.

### Code Refactoring

* Full rewrite of the Go module layout. Concrete resource implementations
  collapsed into one `internal/resources` package; controller plumbing
  consolidated under `internal/k8s`; YAML output split into
  `internal/gatus` (Endpoint, template merge, atomic writer, guarded-DNS
  helpers). Binary moves from `cmd/root.go` to `cmd/gatus-sidecar/main.go`.
* Replaced the hand-rolled watch loop with a `dynamicinformer` +
  `workqueue` controller per resource kind: proper resync, missed-event
  recovery, rate-limited retries, and a single batched flush on initial
  sync instead of one write per resource.
* Output file is now written atomically (tempfile + rename) so Gatus never
  reads a partially-written config.
* Logging audited: noisy per-resource warnings (e.g. "no derivable URL")
  demoted to `debug`; transient retries logged at `warn`; terminal failures
  at `error`.

### Tests

* End-to-end test suite under `test/e2e` (build tag `e2e`) drives a real
  Kind cluster, applies fixtures covering all four resource kinds plus
  template inheritance, and asserts on the generated YAML across create /
  update / delete / disable transitions.
* New GitHub Actions workflow (`.github/workflows/e2e.yaml`) runs the Kind
  E2E suite on every PR.

### Documentation

* README rewritten: dropped stale sections, added concise flag reference,
  documented annotation behavior, URL-derivation rules, and the new prefix /
  multi-value-filter / log-level flags.

## [0.0.15](https://github.com/home-operations/gatus-sidecar/compare/0.0.14...0.0.15) (2026-05-20)


### Bug Fixes

* **deps:** update kubernetes monorepo (v0.36.0 → v0.36.1) ([#60](https://github.com/home-operations/gatus-sidecar/issues/60)) ([a6479d3](https://github.com/home-operations/gatus-sidecar/commit/a6479d33f7c3f49b234c4d2e93c2c3d72df76c6c))


### Miscellaneous Chores

* add mise lockfile and update hooks ([ce2675d](https://github.com/home-operations/gatus-sidecar/commit/ce2675db72afe9c15af0fa347cffc164ddd4ed75))
* add mise, release-please, fix lints ([cc51d07](https://github.com/home-operations/gatus-sidecar/commit/cc51d07e329b39fc9f60a2ec39d1967497b31170))
* consolidation and standardization ([8462956](https://github.com/home-operations/gatus-sidecar/commit/8462956aa49c8f3d944b1dc6d0750526e7548211))
* extend lefthook from .github and split editorconfig ([068665e](https://github.com/home-operations/gatus-sidecar/commit/068665eccde857fe2f64860de491880b4ceeccb9))
* more standardizing ([b1ecb7b](https://github.com/home-operations/gatus-sidecar/commit/b1ecb7be3bc5d58ee255c7fd69a87f99e7fd9928))
