{{/*
Expand the name of the chart.
*/}}
{{- define "aks-mcp.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "aks-mcp.fullname" -}}
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
{{- define "aks-mcp.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "aks-mcp.labels" -}}
helm.sh/chart: {{ include "aks-mcp.chart" . }}
{{ include "aks-mcp.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "aks-mcp.selectorLabels" -}}
app.kubernetes.io/name: {{ include "aks-mcp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "aks-mcp.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "aks-mcp.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create Azure credentials secret name
*/}}
{{- define "aks-mcp.azureSecretName" -}}
{{- if .Values.azure.existingSecret }}
{{- .Values.azure.existingSecret }}
{{- else }}
{{- printf "%s-azure-credentials" (include "aks-mcp.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Generate OAuth redirect URIs
*/}}
{{- define "aks-mcp.oauthRedirectURIs" -}}
{{- $redirects := list -}}
{{- if .Values.oauth.redirectURIs -}}
{{- $redirects = .Values.oauth.redirectURIs -}}
{{- else -}}
{{- if eq .Values.service.type "NodePort" -}}
{{- $redirects = append $redirects (printf "http://localhost:%d/oauth/callback" (.Values.service.nodePort | int)) -}}
{{- end -}}
{{- if .Values.ingress.enabled -}}
{{- range .Values.ingress.hosts -}}
{{- $redirects = append $redirects (printf "https://%s/oauth/callback" .host) -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- join "," $redirects -}}
{{- end }}

{{/*
Generate OAuth CORS origins
*/}}
{{- define "aks-mcp.oauthCorsOrigins" -}}
{{- $origins := list -}}
{{- if .Values.oauth.corsOrigins -}}
{{- $origins = .Values.oauth.corsOrigins -}}
{{- else -}}
{{- if eq .Values.service.type "NodePort" -}}
{{- $origins = append $origins (printf "http://localhost:%d" (.Values.service.nodePort | int)) -}}
{{- end -}}
{{- if .Values.ingress.enabled -}}
{{- range .Values.ingress.hosts -}}
{{- $origins = append $origins (printf "https://%s" .host) -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- join "," $origins -}}
{{- end }}