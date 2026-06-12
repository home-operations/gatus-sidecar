{{- if .Values.rbac.create -}}
apiVersion: rbac.authorization.k8s.io/v1
kind: {{ .Values.rbac.type }}
metadata:
  name: {{ include "gatus-sidecar.fullname" . }}
  {{- if eq .Values.rbac.type "Role" }}
  namespace: {{ .Release.Namespace }}
  {{- end }}
  labels:
    {{- include "gatus-sidecar.labels" . | nindent 4 }}
# Rules are derived from the sidecar kinds you enable (least privilege,
# get/list/watch), plus any rbac.extraRules; see the gatus.rbacRules helper.
rules:
  {{- include "gatus-sidecar.rbacRules" . | nindent 2 }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: {{ .Values.rbac.type }}Binding
metadata:
  name: {{ include "gatus-sidecar.fullname" . }}
  {{- if eq .Values.rbac.type "Role" }}
  namespace: {{ .Release.Namespace }}
  {{- end }}
  labels:
    {{- include "gatus-sidecar.labels" . | nindent 4 }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: {{ .Values.rbac.type }}
  name: {{ include "gatus-sidecar.fullname" . }}
subjects:
  - kind: ServiceAccount
    name: {{ include "gatus-sidecar.serviceAccountName" . }}
    namespace: {{ .Release.Namespace }}
{{- end }}
