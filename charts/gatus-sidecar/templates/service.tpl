apiVersion: v1
kind: Service
metadata:
  name: {{ include "gatus-sidecar.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "gatus-sidecar.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    # Gatus serves the UI/API and /metrics on a single web port; the ServiceMonitor
    # scrapes this same port at /metrics.
    - name: http
      port: {{ .Values.service.port }}
      targetPort: http
      protocol: TCP
  selector:
    {{- include "gatus-sidecar.selectorLabels" . | nindent 4 }}
