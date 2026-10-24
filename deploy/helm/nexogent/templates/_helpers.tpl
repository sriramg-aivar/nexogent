{{/* Base name. */}}
{{- define "nexogent.name" -}}nexogent{{- end -}}

{{/* Common labels (component passed via dict). */}}
{{- define "nexogent.labels" -}}
app.kubernetes.io/part-of: nexogent
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}

{{/* Service account name. */}}
{{- define "nexogent.serviceAccountName" -}}
{{- default "nexogent-sa" .Values.serviceAccount.name -}}
{{- end -}}

{{/* Component selector labels: include this template with a two-item list
     (root context, component name). */}}
{{- define "nexogent.componentSelector" -}}
app.kubernetes.io/name: nexogent
app.kubernetes.io/component: {{ .component }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
{{- end -}}
