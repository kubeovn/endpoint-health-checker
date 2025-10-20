{{/*
获取副本数：
- 节点数未知时返回1
- 节点数<3时返回1
- 节点数>=3时返回1（如需3可改）
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