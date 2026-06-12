{{- if .Values.podDisruptionBudget.enabled }}
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{ include "gatus-sidecar.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "gatus-sidecar.labels" . | nindent 4 }}
spec:
  {{- if .Values.podDisruptionBudget.maxUnavailable }}
  maxUnavailable: {{ .Values.podDisruptionBudget.maxUnavailable }}
  {{- else }}
  minAvailable: {{ .Values.podDisruptionBudget.minAvailable }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "gatus-sidecar.selectorLabels" . | nindent 6 }}
{{- end }}
