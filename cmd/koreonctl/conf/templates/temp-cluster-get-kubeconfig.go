package templates

const ClusterGetKubeconfigText = `
{{- $Master := .KoreOnTemp.NodePool.Master}}

## Inventory for {{.Command}} task.
{{ printf "%.*s" 64 "======================================================================================================================================================" }}
{{"Node Name"|printf "%-*s" 20}}{{"IP"|printf "%-*s" 22}}{{"Private IP"|printf "%-*s" 22}}
{{ printf "%.*s" 64 "======================================================================================================================================================" }}
{{-  range $index, $data := $Master.IP}}
node-{{$index|printf "%-*v" 15}}{{$data|printf "%-*s" 22}}{{(index $Master.PrivateIP $index)|printf "%-*s" 22}}
{{- break }}
{{-  end}}
{{ printf "%.*s" 64 "======================================================================================================================================================" }}
Is this ok [y/n]: `
