{{/*
Get replica count:
- Return 1 when node count is unknown
- Return 1 when node count < 3
- Return 1 when node count >= 3 (can be changed to 3 if needed)
*/}}
{{- define "endpoint-health-checker.replicaCount" -}}
{{- $nodes := 0 -}}
{{- if .Capabilities.APIVersions.Has "v1/Node" -}}
  {{- $nodeList := lookup "v1" "Node" "" "" -}}
  {{- if $nodeList -}}
    {{- $nodes = len $nodeList.items -}}
  {{- end -}}
{{- end -}}
{{- if or (eq $nodes 0) (lt $nodes 3) -}}
1
{{- else -}}
3
{{- end -}}
{{- end -}} 