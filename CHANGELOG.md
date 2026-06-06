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

## [0.1.0](https://github.com/home-operations/gatus-sidecar/compare/0.0.19...0.1.0) (2026-06-06)


### Features

* add build version ([#73](https://github.com/home-operations/gatus-sidecar/issues/73)) ([fdff804](https://github.com/home-operations/gatus-sidecar/commit/fdff804284706302cfa3ce52642b2baa109d7557))
* **mise:** update tool oxfmt (0.52.0 → 0.53.0) ([5d8d78a](https://github.com/home-operations/gatus-sidecar/commit/5d8d78ae5e95327f3205b5f7cf4beb93d6263e61))


### Bug Fixes

* **mise:** update tool go (1.26.3 → 1.26.4) ([c9552f7](https://github.com/home-operations/gatus-sidecar/commit/c9552f7632e3afcb0b307a793a33a6b966ea17a4))


### Miscellaneous Chores

* move mise to mise folder ([405fb5c](https://github.com/home-operations/gatus-sidecar/commit/405fb5c3c74bc328d147fcb4d1cdb09c2e282eeb))
* remove 'Contents' section from README ([ba7f15a](https://github.com/home-operations/gatus-sidecar/commit/ba7f15a1faa5995ac67ee3d0e780e8efd72faae4))
* update mise lockfile ([fb9f3c8](https://github.com/home-operations/gatus-sidecar/commit/fb9f3c82f6da66dd0ddf1a9b12bb4e4d272a744b))
* update rlspls workflow name ([4585d37](https://github.com/home-operations/gatus-sidecar/commit/4585d3749682c2b21358b0390e3ca987647e2502))

## [0.0.19](https://github.com/home-operations/gatus-sidecar/compare/0.0.18...0.0.19) (2026-06-01)


### Bug Fixes

* **mise:** update tool lefthook (2.1.8 → 2.1.9) ([bf4d840](https://github.com/home-operations/gatus-sidecar/commit/bf4d8409a76c3933f7b58992b2f2d2fb6ffd58d5))


### Miscellaneous Chores

* implement oxfmt ([8976e28](https://github.com/home-operations/gatus-sidecar/commit/8976e28671bddd50dc84b49fe8d43cffc1433673))
* remove default draft-pull-request from release-please config ([f0f86dd](https://github.com/home-operations/gatus-sidecar/commit/f0f86dd2285c9b257823c09041b1ebeed16357c0))


### Code Refactoring

* data-driven kind registry, scoped logging, test hardening ([#70](https://github.com/home-operations/gatus-sidecar/issues/70)) ([cce0eb4](https://github.com/home-operations/gatus-sidecar/commit/cce0eb4ff6f24d66ebb22bd18cdd28f7179d4e32))

## [0.0.18](https://github.com/home-operations/gatus-sidecar/compare/0.0.17...0.0.18) (2026-05-21)


### Code Refactoring

* tighten hot paths and tidy tests ([#67](https://github.com/home-operations/gatus-sidecar/issues/67)) ([7916c12](https://github.com/home-operations/gatus-sidecar/commit/7916c12f9ed83933fc05c52a51f172de7e043743))

## [0.0.17](https://github.com/home-operations/gatus-sidecar/compare/0.0.16...0.0.17) (2026-05-21)


### Features

* per-resource path: directive and --probe-paths flag ([#65](https://github.com/home-operations/gatus-sidecar/issues/65)) ([02f2e31](https://github.com/home-operations/gatus-sidecar/commit/02f2e3106b93fd1d5a3879cbde39d23d8ffa1676))

## [0.0.16](https://github.com/home-operations/gatus-sidecar/compare/0.0.15...0.0.16) (2026-05-21)


### Features

* rewrite, path-aware URLs, name prefixes, multi-value filters ([#62](https://github.com/home-operations/gatus-sidecar/issues/62)) ([8373935](https://github.com/home-operations/gatus-sidecar/commit/83739354f9cafeec7c8babcd180983267a4d41da))

## [0.0.15](https://github.com/home-operations/gatus-sidecar/compare/0.0.14...0.0.15) (2026-05-20)


### Bug Fixes

* **deps:** update kubernetes monorepo (v0.36.0 → v0.36.1) ([#60](https://github.com/home-operations/gatus-sidecar/issues/60)) ([a6479d3](https://github.com/home-operations/gatus-sidecar/commit/a6479d33f7c3f49b234c4d2e93c2c3d72df76c6c))


### Miscellaneous Chores

* add mise lockfile and update hooks ([ce2675d](https://github.com/home-operations/gatus-sidecar/commit/ce2675db72afe9c15af0fa347cffc164ddd4ed75))
* add mise, release-please, fix lints ([cc51d07](https://github.com/home-operations/gatus-sidecar/commit/cc51d07e329b39fc9f60a2ec39d1967497b31170))
* consolidation and standardization ([8462956](https://github.com/home-operations/gatus-sidecar/commit/8462956aa49c8f3d944b1dc6d0750526e7548211))
* extend lefthook from .github and split editorconfig ([068665e](https://github.com/home-operations/gatus-sidecar/commit/068665eccde857fe2f64860de491880b4ceeccb9))
* more standardizing ([b1ecb7b](https://github.com/home-operations/gatus-sidecar/commit/b1ecb7be3bc5d58ee255c7fd69a87f99e7fd9928))
