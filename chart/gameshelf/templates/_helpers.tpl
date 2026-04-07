{{/*
Validate required values.
*/}}
{{- define "gameshelf.validate" -}}
{{- if or (empty .Values.adminSecret) (eq .Values.adminSecret "changeme") -}}
{{- fail "adminSecret must be set to a strong secret (not empty or 'changeme')" -}}
{{- end -}}
{{- end }}
{{- include "gameshelf.validate" . }}

{{/*
Expand the name of the chart.
*/}}
{{- define "gameshelf.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "gameshelf.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "gameshelf.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "gameshelf.labels" -}}
helm.sh/chart: {{ include "gameshelf.chart" . }}
{{ include "gameshelf.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "gameshelf.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gameshelf.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
DATABASE_URL construction
*/}}
{{- define "gameshelf.databaseUrl" -}}
{{- if .Values.postgresql.enabled -}}
postgres://{{ .Values.postgresql.auth.username }}:{{ .Values.postgresql.auth.password }}@{{ include "gameshelf.fullname" . }}-postgresql:5432/{{ .Values.postgresql.auth.database }}?sslmode=disable
{{- else -}}
postgres://{{ .Values.externalDatabase.username }}:{{ .Values.externalDatabase.password }}@{{ .Values.externalDatabase.host }}:{{ .Values.externalDatabase.port }}{{ printf "/" }}{{ .Values.externalDatabase.database }}?sslmode=disable
{{- end -}}
{{- end }}

{{/*
REDIS_URL construction
*/}}
{{- define "gameshelf.redisUrl" -}}
{{- if .Values.redis.enabled -}}
redis://{{ include "gameshelf.fullname" . }}-redis-master:6379
{{- else -}}
{{- if .Values.externalRedis.password -}}
redis://:{{ .Values.externalRedis.password }}@{{ .Values.externalRedis.host }}:{{ .Values.externalRedis.port }}
{{- else -}}
redis://{{ .Values.externalRedis.host }}:{{ .Values.externalRedis.port }}
{{- end -}}
{{- end -}}
{{- end }}

{{/*
PostgreSQL host for init container check
*/}}
{{- define "gameshelf.postgresqlHost" -}}
{{- if .Values.postgresql.enabled -}}
{{ include "gameshelf.fullname" . }}-postgresql
{{- else -}}
{{ .Values.externalDatabase.host }}
{{- end -}}
{{- end }}

{{/*
PostgreSQL port for init container check
*/}}
{{- define "gameshelf.postgresqlPort" -}}
{{- if .Values.postgresql.enabled -}}
5432
{{- else -}}
{{ .Values.externalDatabase.port }}
{{- end -}}
{{- end }}
