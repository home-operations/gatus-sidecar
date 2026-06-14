apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "gatus-sidecar.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "gatus-sidecar.labels" . | nindent 4 }}
  {{- with .Values.deploymentAnnotations }}
  # Workload-level annotations — e.g. a Stakater Reloader annotation, which must
  # sit on the Deployment (not the pod) to roll it when the config ConfigMap changes.
  annotations:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "gatus-sidecar.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "gatus-sidecar.labels" . | nindent 8 }}
        {{- with .Values.podLabels }}
        {{- tpl (toYaml .) $ | nindent 8 }}
        {{- end }}
      {{- with .Values.podAnnotations }}
      annotations:
        {{- tpl (toYaml .) $ | nindent 8 }}
      {{- end }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- tpl (toYaml .) $ | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "gatus-sidecar.serviceAccountName" . }}
      automountServiceAccountToken: {{ .Values.serviceAccount.automount }}
      {{- with .Values.priorityClassName }}
      priorityClassName: {{ tpl . $ | quote }}
      {{- end }}
      terminationGracePeriodSeconds: {{ .Values.terminationGracePeriodSeconds }}
      securityContext:
        {{- tpl (toYaml .Values.podSecurityContext) $ | nindent 8 }}
      {{- with .Values.resources }}
      # Pod-level resources (Pod.spec.resources) — one budget shared by the gatus
      # and sidecar containers. Needs Kubernetes 1.34+ (PodLevelResources beta, on
      # by default); the chart's kubeVersion enforces this.
      resources:
        {{- tpl (toYaml .) $ | nindent 8 }}
      {{- end }}
      {{- if .Values.sidecar.enabled }}
      # Native sidecar: an init container with restartPolicy: Always runs for the
      # life of the pod, starting before — and stopping after — the gatus container.
      # It watches the cluster and writes its generated config into the shared
      # /config volume, which gatus then reads.
      initContainers:
        - name: gatus-sidecar
          image: {{ include "gatus-sidecar.sidecarImage" . | quote }}
          imagePullPolicy: {{ .Values.sidecar.image.pullPolicy }}
          restartPolicy: Always
          # Flags are derived from the structured sidecar.* values; see the
          # gatus.sidecarArgs helper and the gatus-sidecar README for the flag set.
          args:
            {{- include "gatus-sidecar.sidecarArgs" . | nindent 12 }}
          {{- with .Values.sidecar.extraEnv }}
          env:
            {{- tpl (toYaml .) $ | nindent 12 }}
          {{- end }}
          securityContext:
            {{- tpl (toYaml .Values.sidecar.securityContext) $ | nindent 12 }}
          volumeMounts:
            - name: config
              mountPath: {{ .Values.gatus.configPath }}
      {{- end }}
      containers:
        - name: gatus
          image: {{ include "gatus-sidecar.image" . | quote }}
          imagePullPolicy: {{ .Values.gatus.image.pullPolicy }}
          securityContext:
            {{- tpl (toYaml .Values.gatus.securityContext) $ | nindent 12 }}
          env:
            # Gatus reads its config from GATUS_CONFIG_PATH; the chart owns this and
            # points it at the shared volume mount. GATUS_WEB_PORT is exported so a
            # user's config can reference ${GATUS_WEB_PORT} for its web.port.
            - name: GATUS_CONFIG_PATH
              value: {{ .Values.gatus.configPath | quote }}
            - name: GATUS_WEB_PORT
              value: {{ .Values.gatus.port | quote }}
            {{- with .Values.gatus.logLevel }}
            - name: GATUS_LOG_LEVEL
              value: {{ . | quote }}
            {{- end }}
            {{- with .Values.gatus.delayStartSeconds }}
            - name: GATUS_DELAY_START_SECONDS
              value: {{ . | quote }}
            {{- end }}
            {{- range $k, $v := .Values.gatus.env }}
            - name: {{ $k }}
              value: {{ tpl (toString $v) $ | quote }}
            {{- end }}
            {{- with .Values.gatus.extraEnv }}
            {{- tpl (toYaml .) $ | nindent 12 }}
            {{- end }}
          {{- with .Values.gatus.envFrom }}
          envFrom:
            {{- tpl (toYaml .) $ | nindent 12 }}
          {{- end }}
          ports:
            - name: http
              containerPort: {{ .Values.gatus.port }}
              protocol: TCP
          {{- with .Values.gatus.startupProbe }}
          startupProbe:
            {{- tpl (toYaml .) $ | nindent 12 }}
          {{- end }}
          {{- with .Values.gatus.livenessProbe }}
          livenessProbe:
            {{- tpl (toYaml .) $ | nindent 12 }}
          {{- end }}
          {{- with .Values.gatus.readinessProbe }}
          readinessProbe:
            {{- tpl (toYaml .) $ | nindent 12 }}
          {{- end }}
          volumeMounts:
            # Shared writable volume: gatus reads its config here and the sidecar
            # writes its generated endpoints here. Backed by an emptyDir (in-memory
            # gatus) or a PVC, depending on persistence.enabled.
            - name: config
              mountPath: {{ .Values.gatus.configPath }}
            {{- range .Values.config.items }}
            # Overlay each BYO ConfigMap file read-only at <configPath>/<path>.
            - name: config-files
              mountPath: {{ printf "%s/%s" $.Values.gatus.configPath .path }}
              subPath: {{ .key }}
              readOnly: true
            {{- end }}
            {{- with .Values.volumeMounts }}
            {{- tpl (toYaml .) $ | nindent 12 }}
            {{- end }}
      volumes:
        - name: config-files
          configMap:
            name: {{ include "gatus-sidecar.configMapName" . }}
        - name: config
          {{- if .Values.persistence.enabled }}
          # Persist gatus.configPath across restarts. The BYO ConfigMap files are
          # subPath-overlaid read-only on top at mount time.
          persistentVolumeClaim:
            claimName: {{ .Values.persistence.existingClaim | default (include "gatus-sidecar.fullname" .) }}
          {{- else }}
          emptyDir: {}
          {{- end }}
        {{- with .Values.volumes }}
        {{- tpl (toYaml .) $ | nindent 8 }}
        {{- end }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- tpl (toYaml .) $ | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- tpl (toYaml .) $ | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- tpl (toYaml .) $ | nindent 8 }}
      {{- end }}
