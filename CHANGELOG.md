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

## [0.3.1](https://github.com/home-operations/gatus-sidecar/compare/0.3.0...0.3.1) (2026-06-15)


### Bug Fixes

* **chart:** align pod securityContext to uid/gid/fsGroup 65532 ([#93](https://github.com/home-operations/gatus-sidecar/issues/93)) ([7c356ff](https://github.com/home-operations/gatus-sidecar/commit/7c356ff6f6fcb33686b699c988be0eadee6ece47))


### Miscellaneous Chores

* **chart:** use the shared curl image for the helm test pod ([#89](https://github.com/home-operations/gatus-sidecar/issues/89)) ([7238948](https://github.com/home-operations/gatus-sidecar/commit/7238948ca0e059a50975faa80741eadf8cf1068f))

## [0.3.0](https://github.com/home-operations/gatus-sidecar/compare/0.2.2...0.3.0) (2026-06-14)


### ⚠ BREAKING CHANGES

* **chart:** use a single pod-level resources key (Pod.spec.resources) ([#87](https://github.com/home-operations/gatus-sidecar/issues/87))

### Features

* **chart:** use a single pod-level resources key (Pod.spec.resources) ([#87](https://github.com/home-operations/gatus-sidecar/issues/87)) ([3ef4971](https://github.com/home-operations/gatus-sidecar/commit/3ef49714724ce451b167f17e3038449beed60358))

## [0.2.2](https://github.com/home-operations/gatus-sidecar/compare/0.2.1...0.2.2) (2026-06-12)


### Features

* **chart:** support Deployment annotations on the gatus-sidecar chart ([#83](https://github.com/home-operations/gatus-sidecar/issues/83)) ([626831a](https://github.com/home-operations/gatus-sidecar/commit/626831a5de61f1f37103571897ca3a4f6450810b))

## [0.2.1](https://github.com/home-operations/gatus-sidecar/compare/0.2.0...0.2.1) (2026-06-12)


### Features

* **chart:** add gatus-sidecar chart bundling Gatus + the sidecar ([#79](https://github.com/home-operations/gatus-sidecar/issues/79)) ([b763322](https://github.com/home-operations/gatus-sidecar/commit/b7633229a8b3f6a5684932bf767824a0d30bba88))


### Bug Fixes

* **deps:** update kubernetes monorepo (v0.36.1 → v0.36.2) ([#80](https://github.com/home-operations/gatus-sidecar/issues/80)) ([c9f1bde](https://github.com/home-operations/gatus-sidecar/commit/c9f1bde6429e215be847f9ca1c4b53eee804a930))


### Miscellaneous Chores

* **mise:** update tool helm (4.2.0 → 4.2.1) ([#82](https://github.com/home-operations/gatus-sidecar/issues/82)) ([e7aa33f](https://github.com/home-operations/gatus-sidecar/commit/e7aa33f5ee4358b94ec07542751886d666fd9833))

## [0.2.0](https://github.com/home-operations/gatus-sidecar/compare/0.1.0...0.2.0) (2026-06-12)


### ⚠ BREAKING CHANGES

* **github-action:** Update action actions/create-github-app-token (v2.2.2 → v3.0.0) ([#48](https://github.com/home-operations/gatus-sidecar/issues/48))
* **github-action:** Update action docker/login-action (v3.7.0 → v4.0.0) ([#43](https://github.com/home-operations/gatus-sidecar/issues/43))
* **github-action:** Update action docker/setup-buildx-action (v3.12.0 → v4.0.0) ([#44](https://github.com/home-operations/gatus-sidecar/issues/44))
* **github-action:** Update action docker/metadata-action (v5.10.0 → v6.0.0) ([#45](https://github.com/home-operations/gatus-sidecar/issues/45))
* **github-action:** Update action docker/build-push-action (v6.19.2 → v7.0.0) ([#46](https://github.com/home-operations/gatus-sidecar/issues/46))
* **github-action:** Update GitHub Artifact Actions ([#40](https://github.com/home-operations/gatus-sidecar/issues/40))
* **github-action:** Update GitHub Artifact Actions (major) ([#28](https://github.com/home-operations/gatus-sidecar/issues/28))
* **github-action:** Update action actions/checkout (v5.0.1 → v6.0.0) ([#23](https://github.com/home-operations/gatus-sidecar/issues/23))
* **github-action:** Update action codex-/return-dispatch (v2.1.0 → v3.0.0) ([#25](https://github.com/home-operations/gatus-sidecar/issues/25))
* **github-action:** Update action golangci/golangci-lint-action (v8.0.0 → v9.0.0) ([#18](https://github.com/home-operations/gatus-sidecar/issues/18))
* **github-action:** Update GitHub Artifact Actions ([#14](https://github.com/home-operations/gatus-sidecar/issues/14))

### Features

* add build version ([#73](https://github.com/home-operations/gatus-sidecar/issues/73)) ([fdff804](https://github.com/home-operations/gatus-sidecar/commit/fdff804284706302cfa3ce52642b2baa109d7557))
* add guarded endpoints ([24d4e87](https://github.com/home-operations/gatus-sidecar/commit/24d4e876aa47312b23d029745c16b9492619e3f5))
* allow gateway-wide annotations ([7427eb3](https://github.com/home-operations/gatus-sidecar/commit/7427eb3dba1e1b26b70e107e1a3f079988b67351))
* allow gateway-wide annotations ([61f3ead](https://github.com/home-operations/gatus-sidecar/commit/61f3ead3f6eb4d2ef4ba27f9ed6ea78e20f6ada3))
* conditionally initialize controllers ([04d7b2a](https://github.com/home-operations/gatus-sidecar/commit/04d7b2ac0f968a028fb809856f9be281b81e674a))
* **container:** update image golang (1.25 → 1.26) ([#34](https://github.com/home-operations/gatus-sidecar/issues/34)) ([22ae55c](https://github.com/home-operations/gatus-sidecar/commit/22ae55cb0b91d7413af13c8a15dcc281636e98db))
* **deps:** update kubernetes packages (v0.34.3 → v0.35.0) ([#30](https://github.com/home-operations/gatus-sidecar/issues/30)) ([0b1c4d4](https://github.com/home-operations/gatus-sidecar/commit/0b1c4d4b553255aa9e1f98905b07ba4482651865))
* **deps:** update module k8s.io/client-go (v0.35.2 → v0.36.0) ([#58](https://github.com/home-operations/gatus-sidecar/issues/58)) ([6dab7a2](https://github.com/home-operations/gatus-sidecar/commit/6dab7a2b446c22519b59088e8fbb4ec7da307017))
* **deps:** update module sigs.k8s.io/gateway-api (v1.3.0 → v1.4.0) ([c9b092e](https://github.com/home-operations/gatus-sidecar/commit/c9b092eedb5912448fe97f2da14a1df1c62a02da))
* **deps:** update module sigs.k8s.io/gateway-api (v1.4.1 → v1.5.0) ([#41](https://github.com/home-operations/gatus-sidecar/issues/41)) ([ca65d79](https://github.com/home-operations/gatus-sidecar/commit/ca65d790b4d0fab8cd7f6f7ebee2dfb7c3572cdb))
* feats for onedr0p ([1fe2307](https://github.com/home-operations/gatus-sidecar/commit/1fe230721c1740646c45507892ed32d935065a51))
* fix up guarded endpoints ([02b6c9e](https://github.com/home-operations/gatus-sidecar/commit/02b6c9e184aa314cda695f8dbdfa7930a5cc2a77))
* implement state manager ([2cfde74](https://github.com/home-operations/gatus-sidecar/commit/2cfde7458e710e15a287f41cf4bb5575edcb05f5))
* implement state manager ([d7fe133](https://github.com/home-operations/gatus-sidecar/commit/d7fe1331f7ffb18a35f45c5a876a228d0cb07fb5))
* **mise:** update tool oxfmt (0.52.0 → 0.53.0) ([5d8d78a](https://github.com/home-operations/gatus-sidecar/commit/5d8d78ae5e95327f3205b5f7cf4beb93d6263e61))
* **mise:** update tool oxfmt (0.53.0 → 0.54.0) ([#75](https://github.com/home-operations/gatus-sidecar/issues/75)) ([28e1908](https://github.com/home-operations/gatus-sidecar/commit/28e19087e33bf5e3105fd730062bbdd3d0d69c27))
* per-resource path: directive and --probe-paths flag ([#65](https://github.com/home-operations/gatus-sidecar/issues/65)) ([02f2e31](https://github.com/home-operations/gatus-sidecar/commit/02f2e3106b93fd1d5a3879cbde39d23d8ffa1676))
* rewrite, path-aware URLs, name prefixes, multi-value filters ([#62](https://github.com/home-operations/gatus-sidecar/issues/62)) ([8373935](https://github.com/home-operations/gatus-sidecar/commit/83739354f9cafeec7c8babcd180983267a4d41da))


### Bug Fixes

* better write detection ([ded9570](https://github.com/home-operations/gatus-sidecar/commit/ded95707a0d7b3b1eb816e93a9af230f1dbe4c9e))
* **ci:** disable sbom-action release asset upload to avoid 403 ([6b3bef6](https://github.com/home-operations/gatus-sidecar/commit/6b3bef644fc809b5a5ffd3b31b619b7a1d6c66e2))
* **deps:** update kubernetes monorepo (v0.36.0 → v0.36.1) ([#60](https://github.com/home-operations/gatus-sidecar/issues/60)) ([a6479d3](https://github.com/home-operations/gatus-sidecar/commit/a6479d33f7c3f49b234c4d2e93c2c3d72df76c6c))
* **deps:** update kubernetes packages (v0.34.1 → v0.34.2) ([#20](https://github.com/home-operations/gatus-sidecar/issues/20)) ([e25bffb](https://github.com/home-operations/gatus-sidecar/commit/e25bffb29a5853f31c19ddbdeb49c9d324bcb02d))
* **deps:** update kubernetes packages (v0.34.2 → v0.34.3) ([#27](https://github.com/home-operations/gatus-sidecar/issues/27)) ([23fe6d4](https://github.com/home-operations/gatus-sidecar/commit/23fe6d4dc079901331042d9a38e481871ce5b76c))
* **deps:** update kubernetes packages (v0.35.0 → v0.35.1) ([#35](https://github.com/home-operations/gatus-sidecar/issues/35)) ([35abd0d](https://github.com/home-operations/gatus-sidecar/commit/35abd0daa1dec6c178002a6d52ed98b1ca9aef18))
* **deps:** update kubernetes packages (v0.35.1 → v0.35.2) ([#42](https://github.com/home-operations/gatus-sidecar/issues/42)) ([6fd850b](https://github.com/home-operations/gatus-sidecar/commit/6fd850b06ce6d05e3b111f67ac430db14b3f28d8))
* **deps:** update module sigs.k8s.io/gateway-api (v1.4.0 → v1.4.1) ([#26](https://github.com/home-operations/gatus-sidecar/issues/26)) ([f2e117b](https://github.com/home-operations/gatus-sidecar/commit/f2e117badadb69de28ed88daa3421d87088ef6bc))
* **deps:** update module sigs.k8s.io/gateway-api (v1.5.0 → v1.5.1) ([#47](https://github.com/home-operations/gatus-sidecar/issues/47)) ([1863fa6](https://github.com/home-operations/gatus-sidecar/commit/1863fa6eba280b54680b0e7a228b3703f0933d05))
* don't skip protocol when domain starts with http ([#39](https://github.com/home-operations/gatus-sidecar/issues/39)) ([49ac534](https://github.com/home-operations/gatus-sidecar/commit/49ac5347de0ea6c5c01ae5dbfc459ca66f29caaf))
* last things ([4662a29](https://github.com/home-operations/gatus-sidecar/commit/4662a29d0b060113b50e0a1d401fdd77f711f4a5))
* **mise:** update tool go (1.26.3 → 1.26.4) ([c9552f7](https://github.com/home-operations/gatus-sidecar/commit/c9552f7632e3afcb0b307a793a33a6b966ea17a4))
* **mise:** update tool lefthook (2.1.8 → 2.1.9) ([bf4d840](https://github.com/home-operations/gatus-sidecar/commit/bf4d8409a76c3933f7b58992b2f2d2fb6ffd58d5))
* opps ([53145b1](https://github.com/home-operations/gatus-sidecar/commit/53145b11d60982efce901879e618e8ddb48aeeda))
* **README:** details ([#32](https://github.com/home-operations/gatus-sidecar/issues/32)) ([bdff1d7](https://github.com/home-operations/gatus-sidecar/commit/bdff1d73a21fa2d7abfac5b1126c6eba650ec5d0))


### Miscellaneous Chores

* a very fat refactor ([d3fde22](https://github.com/home-operations/gatus-sidecar/commit/d3fde22f0da3914147ddd3749c949b084982e5f0))
* add mise lockfile and update hooks ([ce2675d](https://github.com/home-operations/gatus-sidecar/commit/ce2675db72afe9c15af0fa347cffc164ddd4ed75))
* add mise, release-please, fix lints ([cc51d07](https://github.com/home-operations/gatus-sidecar/commit/cc51d07e329b39fc9f60a2ec39d1967497b31170))
* clean up args ([fe80f4c](https://github.com/home-operations/gatus-sidecar/commit/fe80f4cc6c8564da8aea6ec8eef8282acbe65e46))
* consolidation and standardization ([8462956](https://github.com/home-operations/gatus-sidecar/commit/8462956aa49c8f3d944b1dc6d0750526e7548211))
* extend lefthook from .github and split editorconfig ([068665e](https://github.com/home-operations/gatus-sidecar/commit/068665eccde857fe2f64860de491880b4ceeccb9))
* fail hard if you derp args ([1dbb9e8](https://github.com/home-operations/gatus-sidecar/commit/1dbb9e8bed7fb9c0512fe237deeda8044cc05ea0))
* fix imports ([7e55f5d](https://github.com/home-operations/gatus-sidecar/commit/7e55f5db4c6e93b75a0599e62c90555928053320))
* implement oxfmt ([8976e28](https://github.com/home-operations/gatus-sidecar/commit/8976e28671bddd50dc84b49fe8d43cffc1433673))
* **main:** release 0.0.15 ([#61](https://github.com/home-operations/gatus-sidecar/issues/61)) ([e6df63c](https://github.com/home-operations/gatus-sidecar/commit/e6df63cd3850be6c9bce78cf2d990087d1863eac))
* **main:** release 0.0.16 ([#63](https://github.com/home-operations/gatus-sidecar/issues/63)) ([271416d](https://github.com/home-operations/gatus-sidecar/commit/271416d1e33de22d7ba972fea49d079eb49cb869))
* **main:** release 0.0.17 ([#66](https://github.com/home-operations/gatus-sidecar/issues/66)) ([dd04a7a](https://github.com/home-operations/gatus-sidecar/commit/dd04a7ad99b2d90c9b4140eed1c117687178308e))
* **main:** release 0.0.18 ([#68](https://github.com/home-operations/gatus-sidecar/issues/68)) ([294d8a4](https://github.com/home-operations/gatus-sidecar/commit/294d8a4b8b1d572ebbdfc1384223bdc73613a909))
* **main:** release 0.0.19 ([#69](https://github.com/home-operations/gatus-sidecar/issues/69)) ([8e231b4](https://github.com/home-operations/gatus-sidecar/commit/8e231b47de962655d3240364a8bdd1546d983d57))
* **main:** release 0.1.0 ([#71](https://github.com/home-operations/gatus-sidecar/issues/71)) ([cfd8c2a](https://github.com/home-operations/gatus-sidecar/commit/cfd8c2accd5fe612dc2f4271437c5bb12e0165fa))
* more small feats ([0b3bf28](https://github.com/home-operations/gatus-sidecar/commit/0b3bf28af4e36ea3c692bc3c219c5c2221239137))
* more small feats 2 ([9be9d44](https://github.com/home-operations/gatus-sidecar/commit/9be9d447ab02b690b0e896aedd71cc95ee66855a))
* more standardizing ([b1ecb7b](https://github.com/home-operations/gatus-sidecar/commit/b1ecb7be3bc5d58ee255c7fd69a87f99e7fd9928))
* move mise to mise folder ([405fb5c](https://github.com/home-operations/gatus-sidecar/commit/405fb5c3c74bc328d147fcb4d1cdb09c2e282eeb))
* move to scratch ([d89b92c](https://github.com/home-operations/gatus-sidecar/commit/d89b92caf1db71ea0c3b0896931de1f59ac59342))
* opps ([27ee31c](https://github.com/home-operations/gatus-sidecar/commit/27ee31c654478288c955ab4ae19518fe3fbf9e41))
* refactor ([6df5faf](https://github.com/home-operations/gatus-sidecar/commit/6df5faf991e8616ad8a51ef703c2e95111b3e1f4))
* refactor renovate workflow to use GitHub CLI ([ef9afb3](https://github.com/home-operations/gatus-sidecar/commit/ef9afb39e45d659aee4db0b35211782fe9752d90))
* remove 'Contents' section from README ([ba7f15a](https://github.com/home-operations/gatus-sidecar/commit/ba7f15a1faa5995ac67ee3d0e780e8efd72faae4))
* remove default draft-pull-request from release-please config ([f0f86dd](https://github.com/home-operations/gatus-sidecar/commit/f0f86dd2285c9b257823c09041b1ebeed16357c0))
* some nits ([4b66c73](https://github.com/home-operations/gatus-sidecar/commit/4b66c734d1ad7df7b9de79d45a7c07d4112e9508))
* some nits ([e8690f3](https://github.com/home-operations/gatus-sidecar/commit/e8690f3d4c26bce26457297a6f6964b3af994b9d))
* update go ([e68ed68](https://github.com/home-operations/gatus-sidecar/commit/e68ed683521d1ce368c5d035ce12ec00d285c1c4))
* update go 1.26 ([6faafd3](https://github.com/home-operations/gatus-sidecar/commit/6faafd33217900e119376f1f8c8e4c369903d1af))
* update mise lockfile ([fb9f3c8](https://github.com/home-operations/gatus-sidecar/commit/fb9f3c82f6da66dd0ddf1a9b12bb4e4d272a744b))
* update readme ([9955d06](https://github.com/home-operations/gatus-sidecar/commit/9955d061551e690212bca9e5e399ca3b99b258ae))
* update readme ([375d375](https://github.com/home-operations/gatus-sidecar/commit/375d375bbe79c6f2e2705e90d9d377ea68489520))
* update readme ([f0ec36e](https://github.com/home-operations/gatus-sidecar/commit/f0ec36e7ffd73d07d39661215669c6d521679cef))
* update readme ([4090c64](https://github.com/home-operations/gatus-sidecar/commit/4090c642c7b4cf8c87d0aad71edf62dc4600d4fb))
* update rlspls workflow name ([4585d37](https://github.com/home-operations/gatus-sidecar/commit/4585d3749682c2b21358b0390e3ca987647e2502))
* update test workflow ([f3bb3d9](https://github.com/home-operations/gatus-sidecar/commit/f3bb3d9190ad40e45294cb863e12a4555068f946))


### Code Refactoring

* data-driven kind registry, scoped logging, test hardening ([#70](https://github.com/home-operations/gatus-sidecar/issues/70)) ([cce0eb4](https://github.com/home-operations/gatus-sidecar/commit/cce0eb4ff6f24d66ebb22bd18cdd28f7179d4e32))
* tighten hot paths and tidy tests ([#67](https://github.com/home-operations/gatus-sidecar/issues/67)) ([7916c12](https://github.com/home-operations/gatus-sidecar/commit/7916c12f9ed83933fc05c52a51f172de7e043743))


### Continuous Integration

* **github-action:** Update action actions/checkout (v5.0.1 → v6.0.0) ([#23](https://github.com/home-operations/gatus-sidecar/issues/23)) ([73f10e5](https://github.com/home-operations/gatus-sidecar/commit/73f10e58b891d00184a1d2723f2c66c376e46298))
* **github-action:** Update action actions/create-github-app-token (v2.2.2 → v3.0.0) ([#48](https://github.com/home-operations/gatus-sidecar/issues/48)) ([0ba7c8f](https://github.com/home-operations/gatus-sidecar/commit/0ba7c8ffecec7643fdd96c0561a8b645d7e122b3))
* **github-action:** Update action codex-/return-dispatch (v2.1.0 → v3.0.0) ([#25](https://github.com/home-operations/gatus-sidecar/issues/25)) ([8f47a41](https://github.com/home-operations/gatus-sidecar/commit/8f47a41987fa4c28343a60ed2e0686ec9dd55e1a))
* **github-action:** Update action docker/build-push-action (v6.19.2 → v7.0.0) ([#46](https://github.com/home-operations/gatus-sidecar/issues/46)) ([d201e5f](https://github.com/home-operations/gatus-sidecar/commit/d201e5fa66c9c155b3558063f715ba3a5218d320))
* **github-action:** Update action docker/login-action (v3.7.0 → v4.0.0) ([#43](https://github.com/home-operations/gatus-sidecar/issues/43)) ([df026fe](https://github.com/home-operations/gatus-sidecar/commit/df026feee07c1288620fcc42e8d81364a230b2a0))
* **github-action:** Update action docker/metadata-action (v5.10.0 → v6.0.0) ([#45](https://github.com/home-operations/gatus-sidecar/issues/45)) ([5218af4](https://github.com/home-operations/gatus-sidecar/commit/5218af40181e56dbaa469ee31816db3d3cd51d03))
* **github-action:** Update action docker/setup-buildx-action (v3.12.0 → v4.0.0) ([#44](https://github.com/home-operations/gatus-sidecar/issues/44)) ([481275d](https://github.com/home-operations/gatus-sidecar/commit/481275da738718d15fb063d0d6e535dda820d1d4))
* **github-action:** Update action golangci/golangci-lint-action (v8.0.0 → v9.0.0) ([#18](https://github.com/home-operations/gatus-sidecar/issues/18)) ([84b10f0](https://github.com/home-operations/gatus-sidecar/commit/84b10f0d9ce3657bef3bea8ed7f76aa51fa2df21))
* **github-action:** Update GitHub Artifact Actions ([#14](https://github.com/home-operations/gatus-sidecar/issues/14)) ([a759a37](https://github.com/home-operations/gatus-sidecar/commit/a759a371530fa4fc8e2e9ee03c788fa38b0b6a38))
* **github-action:** Update GitHub Artifact Actions ([#40](https://github.com/home-operations/gatus-sidecar/issues/40)) ([f6e8685](https://github.com/home-operations/gatus-sidecar/commit/f6e86851598e97238a7d4f8684736720464cb800))
* **github-action:** Update GitHub Artifact Actions (major) ([#28](https://github.com/home-operations/gatus-sidecar/issues/28)) ([3b1bf00](https://github.com/home-operations/gatus-sidecar/commit/3b1bf00025804508b4c48f8013f676f4127c3f9b))

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
