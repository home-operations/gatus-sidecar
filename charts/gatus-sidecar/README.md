# gatus-sidecar

![Version: 0.0.0](https://img.shields.io/badge/Version-0.0.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.0](https://img.shields.io/badge/AppVersion-0.0.0-informational?style=flat-square)

Deploy Gatus with the home-operations gatus-sidecar for automatic endpoint discovery

**Homepage:** <https://github.com/home-operations/gatus-sidecar>

## Usage

This chart deploys upstream [Gatus](https://github.com/TwiN/gatus) (image
`ghcr.io/twin/gatus`) together with the home-operations
[gatus-sidecar](https://github.com/home-operations/gatus-sidecar) as a native
sidecar. The sidecar watches the cluster (Services, Ingresses, HTTPRoutes, …) and
writes discovered endpoints into a shared config volume that gatus reads, so your
status page stays in sync automatically.

The chart ships as an OCI Helm chart and does **not** render gatus config itself.
Supply a ConfigMap holding your gatus config file(s) and point the chart at it:

```sh
helm install gatus oci://ghcr.io/home-operations/charts/gatus-sidecar \
  --set config.existingConfigMap=gatus-config \
  --set 'config.items[0].key=config.yaml' \
  --set 'config.items[0].path=config.yaml'
```

Each entry in `config.items` is mounted read-only into `gatus.configPath`
(default `/config`) at the given `path`, while the sidecar writes its generated
endpoints alongside them. By default `persistence` is off, so the shared volume
is an `emptyDir` and gatus runs in-memory (set memory storage in your config);
enable `persistence` to back it with a PVC. Configure the sidecar through the
structured `sidecar` values — toggle discovery per kind under `sidecar.kinds`
(see [the gatus-sidecar README](https://github.com/home-operations/gatus-sidecar)
for the full flag set) — and the RBAC rules it needs are derived automatically
from the kinds you enable. The chart sets `GATUS_CONFIG_PATH` and exports
`GATUS_WEB_PORT` (reference it as `web.port: ${GATUS_WEB_PORT}` in your config, or
match `gatus.port`); pass anything else gatus needs (`TZ`, `${VAR}` substitution
values) via `gatus.env`/`gatus.extraEnv`. Sensitive values (e.g. Pushover tokens
in a `buddy.yaml`) are best injected with `gatus.envFrom`. Expose the UI with
either `ingress` or a Gateway API `httpRoute`.

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| home-operations | <contact@home-operations.com> |  |

## Source Code

* <https://github.com/home-operations/gatus-sidecar>

## Requirements

Kubernetes: `>=1.29.0-0`

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity rules for pod scheduling. |
| config.existingConfigMap | string | `""` | REQUIRED: name of a ConfigMap holding your gatus config file(s). |
| config.items | list | `[]` | ConfigMap keys to mount read-only into gatus.configPath; each `{ key, path }` mounts the ConfigMap's `key` at `<configPath>/<path>` (subPath). |
| deploymentAnnotations | object | `{}` | Annotations added to the Deployment (workload) metadata, e.g. `reloader.stakater.com/auto: "true"` so Stakater Reloader rolls the pod when the mounted config ConfigMap changes (Reloader reads the workload, not the pod). |
| fullnameOverride | string | `""` | Override the full release name. |
| gatus.configPath | string | `"/config"` | Directory gatus reads (GATUS_CONFIG_PATH). The shared volume is mounted here; the sidecar writes its generated YAML here and the BYO ConfigMap files are overlaid on top. The chart always sets GATUS_CONFIG_PATH to this. |
| gatus.delayStartSeconds | string | `""` | Delay gatus startup by N seconds (GATUS_DELAY_START_SECONDS). Omitted when empty or 0. |
| gatus.env | object | `{}` | Extra environment variables for the gatus container, as a map (templated), e.g. TZ or `${VAR}` substitution values referenced by your config. |
| gatus.envFrom | list | `[]` | Sources of environment variables for the gatus container (templated), e.g. a Secret holding buddy.yaml Pushover tokens. |
| gatus.extraEnv | list | `[]` | Extra environment variables for the gatus container, as a raw list (templated). |
| gatus.image.digest | string | `""` | Pin the gatus image by digest (sha256:…); when set, overrides the tag. |
| gatus.image.pullPolicy | string | `"IfNotPresent"` | Gatus image pull policy. |
| gatus.image.repository | string | `"ghcr.io/twin/gatus"` | Gatus image repository. |
| gatus.image.tag | string | `"v5.36.0"` | Gatus image tag (Renovate-managed). Also becomes the chart appVersion at package time. |
| gatus.livenessProbe | object | `{"httpGet":{"path":"/health","port":"http"},"initialDelaySeconds":10,"periodSeconds":20}` | Gatus liveness probe. |
| gatus.logLevel | string | `"INFO"` | Gatus log level (GATUS_LOG_LEVEL: DEBUG, INFO, WARN, ERROR). Omitted when empty. |
| gatus.port | int | `8080` | Container port, Service targetPort and probe port. Also exported as GATUS_WEB_PORT so your config can use `web.port: ${GATUS_WEB_PORT}`; gatus's actual listen port comes from its config file, so reference this var or match it. |
| gatus.readinessProbe | object | `{"httpGet":{"path":"/health","port":"http"},"initialDelaySeconds":5,"periodSeconds":10}` | Gatus readiness probe. |
| gatus.resources | object | `{}` | Gatus container resource requests/limits. |
| gatus.securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true}` | Gatus container securityContext (no privilege escalation, read-only root filesystem, drops ALL capabilities). |
| gatus.startupProbe | object | `{}` | Gatus startup probe (optional). Gates liveness/readiness while gatus starts; empty disables it. |
| httpRoute.additionalRules | list | `[]` | Custom rules prepended before the default rule (templated). |
| httpRoute.annotations | object | `{}` | HTTPRoute annotations. |
| httpRoute.apiVersion | string | `""` | HTTPRoute apiVersion; empty defaults to gateway.networking.k8s.io/v1. |
| httpRoute.enabled | bool | `false` | Expose the UI via a Gateway API HTTPRoute (alternative to ingress). |
| httpRoute.filters | list | `[]` | Filters applied to the default rule. |
| httpRoute.hostnames | list | `[]` | Hostnames matched against the Host header (templated). |
| httpRoute.httpsRedirect | bool | `false` | Redirect HTTP→HTTPS (301) instead of routing to the backend (needs HTTP+HTTPS listeners). |
| httpRoute.kind | string | `""` | HTTPRoute kind; empty defaults to HTTPRoute. |
| httpRoute.labels | object | `{}` | HTTPRoute labels. |
| httpRoute.matches | list | `[{"path":{"type":"PathPrefix","value":"/"}}]` | Match conditions for the default rule. |
| httpRoute.parentRefs | list | `[]` | Gateways (and listeners) this route attaches to. |
| imagePullSecrets | list | `[]` | Image pull secrets for private registries. |
| ingress.annotations | object | `{}` | Ingress annotations. |
| ingress.className | string | `""` | IngressClass name. |
| ingress.enabled | bool | `false` | Expose the UI via an Ingress. |
| ingress.hosts | list | `[{"host":"gatus.example.com","paths":[{"path":"/","pathType":"Prefix"}]}]` | Ingress hosts and their paths. |
| ingress.tls | list | `[]` | Ingress TLS configuration. |
| monitoring.serviceMonitor.annotations | object | `{}` | ServiceMonitor annotations. |
| monitoring.serviceMonitor.enabled | bool | `false` | Create a Prometheus Operator ServiceMonitor (requires its CRDs). Scrapes the http port at `path`. |
| monitoring.serviceMonitor.interval | string | `"30s"` | Scrape interval. |
| monitoring.serviceMonitor.labels | object | `{}` | ServiceMonitor labels. |
| monitoring.serviceMonitor.metricRelabelings | list | `[]` | Prometheus metric relabelings. |
| monitoring.serviceMonitor.path | string | `"/metrics"` | Metrics path (served on the gatus web port). |
| monitoring.serviceMonitor.relabelings | list | `[]` | Prometheus relabelings. |
| monitoring.serviceMonitor.scrapeTimeout | string | `"10s"` | Scrape timeout. |
| nameOverride | string | `""` | Override the chart name used in resource names. |
| nodeSelector | object | `{}` | Node selector for pod scheduling. |
| persistence.accessMode | string | `"ReadWriteOnce"` | PVC access mode. |
| persistence.enabled | bool | `false` | Persist gatus.configPath (the sidecar's generated YAML and any sqlite DB) on a PVC. Disabled by default → the shared volume is an emptyDir and gatus runs in-memory (set memory storage in your config). |
| persistence.existingClaim | string | `""` | Use an existing PVC instead of creating one; when set, no PVC is rendered. |
| persistence.size | string | `"1Gi"` | PVC size. |
| persistence.storageClass | string | `""` | StorageClass for the PVC; empty uses the cluster default. |
| podAnnotations | object | `{}` | Annotations added to the pod. |
| podDisruptionBudget.enabled | bool | `false` | Create a PodDisruptionBudget. ⚠️ With a single replica, the default `minAvailable: 1` makes a node drain block until you delete the pod yourself; set `maxUnavailable: 1` instead to let drains proceed. Off by default. |
| podDisruptionBudget.maxUnavailable | string | `""` | Maximum pods that may be unavailable, as a count or percentage; takes precedence over `minAvailable` when set. @schema type: [integer, string] @schema |
| podDisruptionBudget.minAvailable | int | `1` | Minimum pods that must stay available, as a count or percentage. Used unless `maxUnavailable` is set. @schema type: [integer, string] @schema |
| podLabels | object | `{}` | Labels added to the pod. |
| podSecurityContext | object | `{"fsGroup":1000,"fsGroupChangePolicy":"OnRootMismatch","runAsGroup":1000,"runAsNonRoot":true,"runAsUser":1000}` | Pod-level securityContext (runs as non-root uid/gid 1000 with fsGroup so the shared /config volume is writable). |
| priorityClassName | string | `""` | PriorityClass for the pod, so gatus is less likely to be preempted/evicted under node pressure. Empty uses the cluster default. |
| rbac.create | bool | `true` | Create RBAC (a (Cluster)Role + binding) granting the sidecar the access it needs. |
| rbac.extraRules | list | `[]` | Extra policy rules appended to the derived rules. The base rules are generated from the enabled `sidecar.kinds` (least privilege, get/list/watch), so you normally leave this empty. |
| rbac.type | string | `"ClusterRole"` | RBAC scope: ClusterRole (watch all namespaces) or Role (single namespace; pair with the sidecar's `namespace`). |
| replicaCount | int | `1` | Number of gatus replicas. Gatus is backed by a Deployment; with persistence enabled (a single RWO PVC) keep this at 1. |
| service.port | int | `80` | Service port (maps to the gatus web port; /metrics is served here too). |
| service.type | string | `"ClusterIP"` | Service type. |
| serviceAccount.annotations | object | `{}` | Annotations for the ServiceAccount. |
| serviceAccount.automount | bool | `true` | Automount the ServiceAccount API token (on by default: the sidecar needs Kubernetes API access to discover endpoints). |
| serviceAccount.create | bool | `true` | Create a ServiceAccount. |
| serviceAccount.name | string | `""` | ServiceAccount name; generated from the release name if empty. |
| sidecar.annotationConfig | string | `""` | Annotation key for the per-resource YAML config override (--annotation-config); empty uses the sidecar default. |
| sidecar.annotationEnabled | string | `""` | Annotation key for enabling/disabling per-resource processing (--annotation-enabled); empty uses the sidecar default. |
| sidecar.defaultInterval | string | `"1m"` | Default probe interval for generated endpoints (--default-interval). |
| sidecar.enabled | bool | `true` | Run the gatus-sidecar as a native sidecar (init container with restartPolicy: Always). |
| sidecar.extraArgs | list | `[]` | Extra raw flags appended to the sidecar args, e.g. `["--foo=bar"]`. |
| sidecar.extraEnv | list | `[]` | Extra environment variables for the sidecar container, as a raw list (templated). |
| sidecar.gatewayNames | list | `[]` | Gateway name(s) to filter HTTPRoutes (--gateway-name, repeated per entry). |
| sidecar.image.digest | string | `""` | Pin the sidecar image by digest (sha256:…); set by the release pipeline. When set, overrides the tag. |
| sidecar.image.pullPolicy | string | `"IfNotPresent"` | gatus-sidecar image pull policy. |
| sidecar.image.repository | string | `"ghcr.io/home-operations/gatus-sidecar"` | gatus-sidecar image repository. |
| sidecar.image.tag | string | `""` | Overrides the sidecar image tag; defaults to the chart version (the sidecar repo's own release). The release pipeline pins the digest instead. |
| sidecar.ingressClasses | list | `[]` | Ingress class(es) to filter Ingresses (--ingress-class, repeated per entry). |
| sidecar.kinds | object | `{"httproute":{"auto":true,"enable":false,"prefix":""},"ingress":{"auto":false,"enable":false,"prefix":""},"ingressroute":{"auto":false,"enable":false,"prefix":""},"service":{"auto":false,"enable":true,"prefix":""}}` | Per-kind discovery. `enable` turns the kind on; `auto` also auto-creates endpoints for matching resources; `prefix` prepends to generated endpoint names. RBAC rules are derived from whichever kinds are enabled. The default (httproute auto + service enable) mirrors the maintainer's real usage. |
| sidecar.logLevel | string | `"info"` | Sidecar log level (--log-level: debug, info, warn, error). |
| sidecar.namespace | string | `""` | Namespace to watch (--namespace); empty watches all namespaces (requires a ClusterRole). |
| sidecar.output | string | `""` | File the sidecar writes generated YAML to (--output); empty defaults to `<gatus.configPath>/gatus-sidecar.yaml` (in the shared volume). |
| sidecar.probePaths | bool | `true` | Include paths from match rules in probe URLs (--probe-paths); false probes bare hostnames. |
| sidecar.resources | object | `{}` | gatus-sidecar container resource requests/limits. |
| sidecar.securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true}` | gatus-sidecar container securityContext (no privilege escalation, read-only root filesystem, drops ALL capabilities). |
| terminationGracePeriodSeconds | int | `30` | Grace period for a clean shutdown (gatus drains in-flight checks and closes its servers on SIGTERM). |
| tests.image.pullPolicy | string | `"IfNotPresent"` | `helm test` image pull policy. |
| tests.image.repository | string | `"ghcr.io/home-operations/busybox"` | `helm test` pod image; needs a shell with wget (gatus's own image lacks one). |
| tests.image.tag | string | `"1.38.0@sha256:7e2c04dd50ede647bf4a7a4c8dbd629dd4971cd139b9b88fb22bfc3c7a6c13df"` | `helm test` image, pinned as `tag@sha256:digest` so Renovate bumps the tag and its digest together. |
| tolerations | list | `[]` | Tolerations for pod scheduling. |
| volumeMounts | list | `[]` | Additional volume mounts on the gatus container. |
| volumes | list | `[]` | Additional volumes on the Deployment pod. |

---

_This README is generated by [helm-docs](https://github.com/norwoodj/helm-docs) from `Chart.yaml` and `values.yaml`. Edit those (or `README.md.gotmpl`) and run `mise run generate`._
