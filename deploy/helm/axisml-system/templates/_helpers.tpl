{{/*
Chart name, truncated to 63 chars.
*/}}
{{- define "axisml.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fullname: defaults to the release name (expected to be `axisml`).
*/}}
{{- define "axisml.fullname" -}}
{{- .Values.fullnameOverride | default .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Chart label value.
*/}}
{{- define "axisml.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to every resource.
*/}}
{{- define "axisml.labels" -}}
helm.sh/chart: {{ include "axisml.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: axisml
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Per-component labels.
Usage: include "axisml.componentLabels" (dict "context" . "component" "platform")
*/}}
{{- define "axisml.componentLabels" -}}
{{ include "axisml.labels" .context }}
app.kubernetes.io/name: {{ .component }}
app.kubernetes.io/instance: {{ .context.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Per-component selector labels (subset used in matchLabels).
Usage: include "axisml.componentSelectorLabels" (dict "context" . "component" "platform")
*/}}
{{- define "axisml.componentSelectorLabels" -}}
app.kubernetes.io/name: {{ .component }}
app.kubernetes.io/instance: {{ .context.Release.Name }}
{{- end }}

{{/*
Image helper: resolves global registry override + tag defaulting to appVersion.
Usage: include "axisml.image" (dict "imageRoot" .Values.platform.image "global" .Values.global "chart" .Chart)
*/}}
{{- define "axisml.image" -}}
{{- $registry := .imageRoot.registry | default .global.imageRegistry | default "" -}}
{{- $tag := .imageRoot.tag | default .chart.AppVersion -}}
{{- if $registry -}}
{{- printf "%s/%s:%s" $registry .imageRoot.repository $tag -}}
{{- else -}}
{{- printf "%s:%s" .imageRoot.repository $tag -}}
{{- end -}}
{{- end }}

{{/*
Database host helper.
*/}}
{{- define "axisml.databaseHost" -}}
{{- if .Values.database.enabled -}}
{{- .Values.database.fullnameOverride | default "axisml-database" -}}
{{- else -}}
{{- .Values.externalDatabase.host -}}
{{- end -}}
{{- end }}

{{/*
Database port helper.
*/}}
{{- define "axisml.databasePort" -}}
{{- if .Values.database.enabled -}}
{{- .Values.database.port | default 5432 -}}
{{- else -}}
{{- .Values.externalDatabase.port | default 5432 -}}
{{- end -}}
{{- end }}

{{/*
wait-for-database initContainer. Polls TCP reachability of the database
Service until it accepts connections; the bitnami/postgresql sub-chart's
pg_isready readiness probe gates Service endpoints, so a successful TCP
connect implies the DB is ready to accept queries.

Without this gate, the compute-service Deployment + bootstrap post-install Job
crash-loop while the DB is still starting (image pull + initdb on a
fresh PVC), and the bootstrap Job exhausts its backoffLimit before the
DB ever becomes reachable. Caller passes the chart context as `.`.
*/}}
{{- define "axisml.waitForDatabase" -}}
- name: wait-for-database
  image: busybox:1.36.1
  imagePullPolicy: IfNotPresent
  env:
    - name: DATABASE_HOST
      value: {{ include "axisml.databaseHost" . | quote }}
    - name: DATABASE_PORT
      value: {{ include "axisml.databasePort" . | quote }}
  command:
    - sh
    - -c
    - |
      echo "wait-for-database: $DATABASE_HOST:$DATABASE_PORT"
      until nc -z "$DATABASE_HOST" "$DATABASE_PORT"; do
        echo "wait-for-database: still unavailable; sleeping 2s"
        sleep 2
      done
      echo "wait-for-database: $DATABASE_HOST:$DATABASE_PORT is reachable"
{{- end }}
