# gatus-sidecar

> A Kubernetes sidecar for [Gatus](https://github.com/TwiN/gatus) — turns Ingress, Service, Gateway API HTTPRoute, and Traefik IngressRoute resources into Gatus endpoint configuration, automatically.

[![CI](https://github.com/home-operations/gatus-sidecar/actions/workflows/tests.yaml/badge.svg)](https://github.com/home-operations/gatus-sidecar/actions/workflows/tests.yaml)
[![E2E](https://github.com/home-operations/gatus-sidecar/actions/workflows/e2e.yaml/badge.svg)](https://github.com/home-operations/gatus-sidecar/actions/workflows/e2e.yaml)
[![Lint](https://github.com/home-operations/gatus-sidecar/actions/workflows/lint.yaml/badge.svg)](https://github.com/home-operations/gatus-sidecar/actions/workflows/lint.yaml)
[![Release](https://img.shields.io/github/v/release/home-operations/gatus-sidecar)](https://github.com/home-operations/gatus-sidecar/releases)
[![License](https://img.shields.io/github/license/home-operations/gatus-sidecar)](LICENSE)

---

## Contents

- [Why](#why)
- [Resource support](#resource-support)
- [Quick start](#quick-start)
- [Deployment](#deployment)
- [Configuration](#configuration)
  - [Flag reference](#flag-reference)
  - [Annotations](#annotations)
  - [Template merging](#template-merging)
  - [URL derivation](#url-derivation)
  - [Guarded probes](#guarded-probes)
- [Examples](#examples)
- [Development](#development)
- [Architecture](#architecture)
- [License](#license)

---

## Why

Maintaining a Gatus config by hand stops scaling once you have more than a handful of services. This sidecar watches the cluster, derives an endpoint per resource, and writes a YAML file Gatus hot-reloads — no restarts, no template hell, no drift.

- 🪶 **Single binary, scratch image** — minimal footprint, no runtime deps.
- 🔄 **Hot reload** — atomic file writes; Gatus picks up changes automatically.
- 🎯 **Auto or opt-in** — discover every resource, or only ones annotated explicitly.
- 🧬 **Annotation inheritance** — Gateway → HTTPRoute, IngressClass → Ingress.
- 🛣️ **Path-aware URLs** — extracts paths from Ingress rules and HTTPRoute matches.
- 🏷️ **Per-kind name prefixes** — keep an Ingress and a Service with the same name from colliding.

## Resource support

| Resource | Group / Version | Parent (annotation inheritance) | URL shape |
|---|---|---|---|
| **Ingress** | `networking.k8s.io/v1` | `IngressClass` | `http(s)://<host><path>` |
| **Service** | `v1` | — | `<proto>://<name>.<namespace>.svc:<port>` |
| **HTTPRoute** | `gateway.networking.k8s.io/v1` | `Gateway` | `https://<host><path>` |
| **IngressRoute** | `traefik.io/v1alpha1` | — | `http(s)://<host><path>` |

## Quick start

```bash
docker pull ghcr.io/home-operations/gatus-sidecar:latest
```

Minimum-viable local run against your current kubeconfig:

```bash
gatus-sidecar \
  --auto-ingress \
  --auto-service \
  --output=./gatus-sidecar.yaml
```

…then point Gatus at the generated file.

## Deployment

Run alongside Gatus, sharing a config volume. The sidecar writes
`/config/gatus-sidecar.yaml`; configure Gatus to include it as a watched
sub-config.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gatus
spec:
  template:
    spec:
      serviceAccountName: gatus-sidecar
      containers:
        - name: gatus
          image: ghcr.io/twin/gatus:latest
          volumeMounts:
            - { name: gatus-config, mountPath: /config }
        - name: gatus-sidecar
          image: ghcr.io/home-operations/gatus-sidecar:latest
          args:
            - --auto-ingress
            - --auto-service
            - --auto-httproute
            - --gateway-name=production
          volumeMounts:
            - { name: gatus-config, mountPath: /config }
      volumes:
        - { name: gatus-config, emptyDir: {} }
```

Minimum RBAC for the controllers you enable:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata: { name: gatus-sidecar }
rules:
  - apiGroups: [""]
    resources: ["services"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["networking.k8s.io"]
    resources: ["ingresses", "ingressclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["gateway.networking.k8s.io"]
    resources: ["httproutes", "gateways"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["traefik.io"]
    resources: ["ingressroutes"]
    verbs: ["get", "list", "watch"]
```

## Configuration

### Flag reference

#### Discovery modes

One per resource type. With no `--enable-*`/`--auto-*` flag set, every kind runs in **annotation-only** mode (resources must opt in).

| Flag | Effect |
|---|---|
| `--auto-ingress` | Emit an endpoint for every in-scope Ingress. |
| `--auto-service` | Emit an endpoint for every in-scope Service. |
| `--auto-httproute` | Emit an endpoint for every in-scope HTTPRoute. |
| `--auto-ingressroute` | Emit an endpoint for every Traefik IngressRoute. |
| `--enable-ingress` `--enable-service` `--enable-httproute` `--enable-ingressroute` | Watch the kind, but only emit for resources annotated `gatus.home-operations.com/enabled: "true"`. |

#### Filtering

| Flag | Repeatable? | Effect |
|---|---|---|
| `--namespace` | no | Watch a single namespace (empty = all). |
| `--ingress-class` | **yes** | Only Ingresses whose class is in the set are emitted. |
| `--gateway-name` | **yes** | Only HTTPRoutes referencing a Gateway in the set are emitted. |

> Repeatable flags can be passed multiple times: `--ingress-class=nginx --ingress-class=traefik` matches either.

#### Naming

Use these to disambiguate endpoints across resource kinds — Gatus rejects duplicate `name`s, so prefix per-kind whenever an Ingress and a Service might share a name.

| Flag | Prepended to endpoint name |
|---|---|
| `--prefix-ingress` | Ingress endpoints |
| `--prefix-service` | Service endpoints |
| `--prefix-httproute` | HTTPRoute endpoints |
| `--prefix-ingressroute` | IngressRoute endpoints |

#### Output & runtime

| Flag | Default | Description |
|---|---|---|
| `--output` | `/config/gatus-sidecar.yaml` | Destination YAML file (written atomically). |
| `--default-interval` | `1m` | Probe interval when not overridden by an annotation. |
| `--annotation-config` | `gatus.home-operations.com/endpoint` | Annotation key for YAML template overrides. |
| `--annotation-enabled` | `gatus.home-operations.com/enabled` | Annotation key for the on/off gate. |
| `--log-level` | `info` | `debug` \| `info` \| `warn` \| `error`. |

### Annotations

| Annotation | Value | Effect |
|---|---|---|
| `gatus.home-operations.com/enabled` | `"true"` / `"1"` | Force-include this resource in annotation-only mode, or keep it in `--auto-*` mode. |
| `gatus.home-operations.com/enabled` | anything else | Exclude this resource even when `--auto-*` is set. |
| `gatus.home-operations.com/endpoint` | YAML fragment | Merged into the generated endpoint (see below). |

### Template merging

The `endpoint` annotation accepts any subset of a Gatus endpoint. Known keys
are merged into typed fields; unknown keys are inlined verbatim — so
`alerts:`, `headers:`, `body:`, etc., all work out of the box.

| Template key | Behavior |
|---|---|
| `name`, `group`, `url`, `interval` | Override the field. |
| `conditions` | Replace the default conditions. Accepts string or list. |
| `dns`, `client`, `ui` | Deep-merged into the field's map. |
| `guarded` | If present, switches the endpoint to a DNS probe. |
| _anything else_ | Inlined into the YAML output as-is. |

For resources with a parent (HTTPRoute → Gateway, Ingress → IngressClass) the
**parent's annotation is merged first; the child wins on conflicts** for
scalars, deep-merges for nested maps. Use the parent for common alerting and
the child for per-route conditions.

### URL derivation

| Resource | Host | Scheme | Path |
|---|---|---|---|
| **Ingress** | First rule with `host` | `https` if TLS covers that host, else `http` | First non-`/` path under the first rule's HTTP block |
| **HTTPRoute** | `spec.hostnames[0]` | `https` (always) | First `Exact`/`PathPrefix` match value (regex matches skipped) |
| **Service** | `<name>.<namespace>.svc` | First port's protocol, lowercased (`tcp://`, `udp://`) | — |
| **IngressRoute** | First `Host(\`...\`)` in a route's `match` | `https` if `spec.tls` is set, else `http` | First `Path(\`...\`)` / `PathPrefix(\`...\`)` in the same `match` |

> Trivial paths (empty, `/`, non-rooted) are dropped so the URL stays bare.
>
> If the host already starts with `http://` or `https://` the scheme is preserved verbatim (useful for explicit override).

### Guarded probes

Set `guarded: true` in a template to replace the HTTP probe with a DNS query
against `1.1.1.1` for the resource's hostname. Useful when the sidecar pod
can't actually reach the service (split-horizon DNS, external-only ingress)
but you still want to know DNS is resolving.

```yaml
metadata:
  annotations:
    gatus.home-operations.com/endpoint: |
      guarded: true
```

## Examples

### Inherit alerts from a Gateway, override per route

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: prod
  annotations:
    gatus.home-operations.com/endpoint: |
      alerts:
        - type: slack
          webhook-url: https://hooks.slack.com/...
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: api
  annotations:
    gatus.home-operations.com/endpoint: |
      interval: 30s
      conditions:
        - "[STATUS] == 200"
        - "[RESPONSE_TIME] < 500"
spec:
  parentRefs: [{ name: prod }]
  hostnames: ["api.example.com"]
  rules:
    - matches: [{ path: { type: PathPrefix, value: "/v1" } }]
```

Generated endpoint URL: `https://api.example.com/v1` — inherits the Gateway's
`alerts`, uses the route's own interval/conditions.

### Disambiguate same-named resources

```bash
gatus-sidecar --auto-ingress --auto-service \
  --prefix-ingress=ing/ --prefix-service=svc/
```

An Ingress and a Service both named `web-app` produce `ing/web-app` and
`svc/web-app` instead of a Gatus startup error.

### Watch multiple gateway classes at once

```bash
gatus-sidecar --auto-httproute \
  --gateway-name=public --gateway-name=internal
```

## Development

```bash
go build -o gatus-sidecar ./cmd/gatus-sidecar
go test ./...                              # unit tests
mise run lint                              # golangci-lint
go test -tags e2e ./test/e2e/...           # Kind-based E2E (needs KUBECONFIG)
```

The project pins tool versions via [mise](https://mise.jdx.dev). Running
`mise install` resolves Go, golangci-lint, lefthook, yamlfmt, and zizmor.

## Architecture

```
cmd/gatus-sidecar/       Entry point
internal/config/         CLI flag parsing & validation
internal/gatus/          Endpoint type, template merge, atomic YAML writer
internal/k8s/            Dynamic-informer controller, Resource interface
internal/resources/      Ingress / Service / HTTPRoute / IngressRoute
test/e2e/                Kind-driven end-to-end suite (build tag: e2e)
```

One `Controller` runs per enabled resource kind. Each uses a
`dynamicinformer` to watch its GVR and feeds a shared `gatus.Writer` with the
merged endpoint set; the writer renders YAML to disk via tempfile + rename,
so Gatus never reads a partial file.

```
   ┌──────────────────┐                          ┌─────────────────┐
   │  Kubernetes API  │ ──── informers/watches ──▶│  gatus-sidecar  │
   │                  │                           │  ┌────────────┐ │
   │  Ingress         │                           │  │ Controller │ │── reconcile ──┐
   │  Service         │                           │  └────────────┘ │               │
   │  HTTPRoute       │                           │  ┌────────────┐ │               ▼
   │  IngressRoute    │                           │  │  Writer    │ ── atomic ──▶ /config/gatus-sidecar.yaml
   │  IngressClass    │                           │  └────────────┘ │               │
   │  Gateway         │                           └─────────────────┘               │
   └──────────────────┘                                                              ▼
                                                                            ┌──────────────┐
                                                                            │    Gatus     │ ◀── hot-reload
                                                                            └──────────────┘
```

## License

MIT — see [LICENSE](LICENSE).
