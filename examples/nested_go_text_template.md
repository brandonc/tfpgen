```
{{ define "Attrs" }}
	"{{.TfName}}": {
		MarkdownDescription: "{{ .Description }}",
		Type:                types.{{ .DataType }},
	},
	{{ if .Attributes }}
	// Nested Attributes:
	{{- range $attribute := .Attributes }}{{ template "Attrs" $attribute }}{{- end}}
	{{ end }}
{{ end }}
```
