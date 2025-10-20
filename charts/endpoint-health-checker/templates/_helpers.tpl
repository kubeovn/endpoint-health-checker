{{/*
Get IP-addresses of master nodes
*/}}
{{- define "endpoint-health-checker.masterNodeIPs" -}}
{{- $nodes := lookup "v1" "Node" "" "" -}}
{{- $ips := list -}}
{{- range $node := $nodes.items -}}
  {{- if hasKey $node.metadata.labels "node-role.kubernetes.io/master" -}}
    {{- range $address := $node.status.addresses -}}
      {{- if eq $address.type "InternalIP" -}}
        {{- $ips = append $ips $address.address -}}
      {{- end -}}
    {{- end -}}
  {{- else if hasKey $node.metadata.labels "node-role.kubernetes.io/control-plane" -}}
    {{- range $address := $node.status.addresses -}}
      {{- if eq $address.type "InternalIP" -}}
        {{- $ips = append $ips $address.address -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- if and (eq (len $ips) 0) (not $.Values.MASTER_NODES) -}}
  {{- fail "No master nodes found. Please ensure master/control-plane nodes are properly labeled." -}}
{{- end -}}
{{ join "," $ips }}
{{- end -}}

{{/*
Number of master nodes
*/}}
{{- define "endpoint-health-checker.masterNodeCount" -}}
  {{- len (split "," (.Values.MASTER_NODES | default (include "endpoint-health-checker.masterNodeIPs" .))) }}
{{- end -}}

{{/*
Get replica count based on master nodes:
- Return 1 when master node count < 3
- Return 2 when master node count >= 3
*/}}
{{- define "endpoint-health-checker.replicaCount" -}}
{{- $masterCount := include "endpoint-health-checker.masterNodeCount" . | int -}}
{{- if lt $masterCount 3 -}}
1
{{- else -}}
2
{{- end -}}
{{- end -}} 