{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "nacos.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "nacos.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "nacos.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Generate a random token if not set.
*/}}
{{- define "nacos.auth.token" -}}
{{- if .Values.nacos.auth.enable}}
    {{- if not (empty .Values.nacos.auth.token) }}
        {{- .Values.nacos.auth.token -}}
    {{- else -}}
        {{- randAlphaNum 64 -}}
    {{- end -}}
{{- end -}}
{{- end -}}


{{/*
Generate a random identity key if not set.
*/}}
{{- define "nacos.auth.identityKey" -}}
{{- if .Values.nacos.auth.enable}}
    {{- if not (empty .Values.nacos.auth.identityKey) }}
        {{- .Values.nacos.auth.identityKey -}}
    {{- else -}}
        {{- randAlphaNum 16  -}}
    {{- end -}}
{{- end -}}
{{- end -}}

{{/*
Generate a random identity value if not set.
*/}}
{{- define "nacos.auth.identityValue" -}}
{{- if .Values.nacos.auth.enable}}
    {{- if not (empty .Values.nacos.auth.identityValue) }}
        {{- .Values.nacos.auth.identityValue -}}
    {{- else -}}
        {{- randAlphaNum 16  -}}
    {{- end -}}
{{- end -}}
{{- end -}}



{{/*
Common labels
*/}}
{{- define "nacos.labels" -}}
app.kubernetes.io/name: {{ include "nacos.name" . }}
helm.sh/chart: {{ include "nacos.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}
