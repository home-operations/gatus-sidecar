apiVersion: v1
kind: Pod
metadata:
  name: {{ include "gatus-sidecar.fullname" . }}-test-connection
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "gatus-sidecar.labels" . | nindent 4 }}
  annotations:
    helm.sh/hook: test
    # Recreate on each run; keep the pod on failure so `helm test --logs` (and a
    # manual `kubectl logs`) can show what happened.
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
spec:
  restartPolicy: Never
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    runAsGroup: 1000
    seccompProfile:
      type: RuntimeDefault
  containers:
    - name: connection
      image: {{ include "gatus-sidecar.testImage" . | quote }}
      imagePullPolicy: {{ .Values.tests.image.pullPolicy }}
      securityContext:
        allowPrivilegeEscalation: false
        readOnlyRootFilesystem: true
        capabilities:
          drop:
            - ALL
      # Gatus serves /health on the web port; this checks that the Service routes
      # to a running, listening pod. wget writes to stdout (-O-) so the rootfs stays
      # read-only; a non-2xx or refused connection exits non-zero and fails the test.
      command:
        - wget
      args:
        - -q
        - -O-
        - http://{{ include "gatus-sidecar.fullname" . }}:{{ .Values.service.port }}/health
