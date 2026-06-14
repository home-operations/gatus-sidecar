{{/*
Expand the name of the chart.
*/}}
{{- define "gatus-sidecar.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name (truncated to the 63-char DNS limit).
*/}}
{{- define "gatus-sidecar.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Chart name and version as used by the chart label.
*/}}
{{- define "gatus-sidecar.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "gatus-sidecar.labels" -}}
helm.sh/chart: {{ include "gatus-sidecar.chart" . }}
{{ include "gatus-sidecar.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "gatus-sidecar.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gatus-sidecar.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Service account name to use.
*/}}
{{- define "gatus-sidecar.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "gatus-sidecar.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Gatus container image reference. A digest pins immutably and wins when set;
otherwise it's repository:tag, with tag defaulting to the chart appVersion (the
gatus version this chart was packaged against). Renovate bumps the tag/digest.
*/}}
{{- define "gatus-sidecar.image" -}}
{{- if .Values.gatus.image.digest -}}
{{- printf "%s@%s" .Values.gatus.image.repository .Values.gatus.image.digest -}}
{{- else -}}
{{- printf "%s:%s" .Values.gatus.image.repository (.Values.gatus.image.tag | default .Chart.AppVersion) -}}
{{- end -}}
{{- end }}

{{/*
gatus-sidecar container image reference. A digest pins immutably and wins when
set (the release pipeline fills it with the published image's digest); otherwise
it's repository:tag, with tag defaulting to the chart version — the sidecar repo's
own release version, since the chart and the sidecar ship together.
*/}}
{{- define "gatus-sidecar.sidecarImage" -}}
{{- if .Values.sidecar.image.digest -}}
{{- printf "%s@%s" .Values.sidecar.image.repository .Values.sidecar.image.digest -}}
{{- else -}}
{{- printf "%s:%s" .Values.sidecar.image.repository (.Values.sidecar.image.tag | default .Chart.Version) -}}
{{- end -}}
{{- end }}

{{/*
Image for the `helm test` connection pod (gatus's own image lacks a shell with
wget). The tag is pinned as `version@sha256:digest`, so Renovate updates the
version and digest together.
*/}}
{{- define "gatus-sidecar.testImage" -}}
{{- $img := .Values.tests.image -}}
{{- printf "%s:%s" $img.repository $img.tag -}}
{{- end }}

{{/*
Name of the ConfigMap holding the gatus config files. This is required — the
chart does not render config — so callers must set config.existingConfigMap.
*/}}
{{- define "gatus-sidecar.configMapName" -}}
{{- tpl (required "config.existingConfigMap is required: provide a ConfigMap with your gatus config files" .Values.config.existingConfigMap) $ -}}
{{- end }}

{{/*
Path the sidecar writes its generated YAML to. Defaults to
<gatus.configPath>/gatus-sidecar.yaml when sidecar.output is empty, so the file
lands in the shared volume gatus reads.
*/}}
{{- define "gatus-sidecar.sidecarOutput" -}}
{{- .Values.sidecar.output | default (printf "%s/gatus-sidecar.yaml" .Values.gatus.configPath) -}}
{{- end }}

{{/*
sidecar flag list, rendered as a YAML sequence, derived from the structured
sidecar.* values (see internal/config/config.go for the authoritative flags).
Always emits --output, --default-interval, --probe-paths and --log-level; emits
--namespace / --annotation-config / --annotation-enabled only when non-empty;
emits --gateway-name / --ingress-class once per list item; per kind emits
--enable-<kind> / --auto-<kind> / --prefix-<kind> as configured; then appends
sidecar.extraArgs verbatim.
*/}}
{{- define "gatus-sidecar.sidecarArgs" -}}
{{- $s := .Values.sidecar -}}
- --output={{ include "gatus-sidecar.sidecarOutput" . }}
- --default-interval={{ $s.defaultInterval }}
- --probe-paths={{ $s.probePaths }}
- --log-level={{ $s.logLevel }}
{{- with $s.namespace }}
- --namespace={{ . }}
{{- end }}
{{- with $s.annotationConfig }}
- --annotation-config={{ . }}
{{- end }}
{{- with $s.annotationEnabled }}
- --annotation-enabled={{ . }}
{{- end }}
{{- range $s.gatewayNames }}
- --gateway-name={{ . }}
{{- end }}
{{- range $s.ingressClasses }}
- --ingress-class={{ . }}
{{- end }}
{{- range $kind := list "ingress" "httproute" "service" "ingressroute" }}
{{- $kc := index $s.kinds $kind }}
{{- if $kc.enable }}
- --enable-{{ $kind }}
{{- end }}
{{- if $kc.auto }}
- --auto-{{ $kind }}
{{- end }}
{{- with $kc.prefix }}
- --prefix-{{ $kind }}={{ . }}
{{- end }}
{{- end }}
{{- range $s.extraArgs }}
- {{ . }}
{{- end }}
{{- end }}

{{/*
RBAC policy rules, rendered as a YAML sequence, DERIVED from which sidecar kinds
are enabled (enable OR auto) — least privilege, get/list/watch only. Appends
rbac.extraRules. Renders an empty list ("[]") when no kind is enabled and no
extra rules are set.
*/}}
{{- define "gatus-sidecar.rbacRules" -}}
{{- $s := .Values.sidecar -}}
{{- $rules := list -}}
{{- if or (index $s.kinds "service").enable (index $s.kinds "service").auto -}}
{{- $rules = append $rules (dict "apiGroups" (list "") "resources" (list "services") "verbs" (list "get" "list" "watch")) -}}
{{- end -}}
{{- if or (index $s.kinds "ingress").enable (index $s.kinds "ingress").auto -}}
{{- $rules = append $rules (dict "apiGroups" (list "networking.k8s.io") "resources" (list "ingresses" "ingressclasses") "verbs" (list "get" "list" "watch")) -}}
{{- end -}}
{{- if or (index $s.kinds "httproute").enable (index $s.kinds "httproute").auto -}}
{{- $rules = append $rules (dict "apiGroups" (list "gateway.networking.k8s.io") "resources" (list "httproutes" "gateways") "verbs" (list "get" "list" "watch")) -}}
{{- end -}}
{{- if or (index $s.kinds "ingressroute").enable (index $s.kinds "ingressroute").auto -}}
{{- $rules = append $rules (dict "apiGroups" (list "traefik.io") "resources" (list "ingressroutes") "verbs" (list "get" "list" "watch")) -}}
{{- end -}}
{{- range .Values.rbac.extraRules -}}
{{- $rules = append $rules . -}}
{{- end -}}
{{- if $rules -}}
{{- toYaml $rules -}}
{{- else -}}
[]
{{- end -}}
{{- end }}
