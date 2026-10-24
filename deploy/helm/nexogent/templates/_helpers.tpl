{{/* Chart name. */}}
{{- define "nexogent.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Fully qualified app name. */}}
{{- define "nexogent.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/* Common labels. */}}
{{- define "nexogent.labels" -}}
app.kubernetes.io/name: {{ include "nexogent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: nexogent
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}

{{/* Selector labels. */}}
{{- define "nexogent.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nexogent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* Service account name. */}}
{{- define "nexogent.serviceAccountName" -}}
{{- default (printf "%s-sa" (include "nexogent.fullname" .)) .Values.serviceAccount.name -}}
{{- end -}}
