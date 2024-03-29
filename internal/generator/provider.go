package generator

import (
	"fmt"

	"github.com/brandonc/tfpgen/internal/config"
	"github.com/getkin/kin-openapi/openapi3"
)

type ProviderGenerator struct {
	Config *config.Config
	Doc    *openapi3.T
}

var _ Generator = (*ProviderGenerator)(nil)

type ProviderResourceData struct {
	DefaultEndpoint string
	PackageName     string
	ProviderName    string
	Resources       []*config.TerraformResource
	DataSources     []*config.TerraformResource
}

func (g *ProviderGenerator) Template() string {
	return `// Code generated by tfpgen; DO NOT EDIT.
package {{ .PackageName }}

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Provider struct {
	Configured bool
	Version string
}

type providerData struct {
	ApiToken types.String ` + "`tfsdk:\"api_token\"`" + `
	Endpoint types.String ` + "`tfsdk:\"endpoint\"`" + `
}

func (p *Provider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "{{ .ProviderName }}"
	resp.Version = p.Version
}

func (p *Provider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The HTTP API endpoint for the provider",
				Optional:            true,
			},
			"api_token": schema.StringAttribute{
				MarkdownDescription: "The HTTP API token, sent as Authorization: Bearer header",
				Optional:            true,
			},
		},
	}
}

func (p *Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Endpoint.IsNull() {
		data.Endpoint = types.StringValue("{{ .DefaultEndpoint }}")
	}

	// If the upstream provider SDK or HTTP client requires configuration, such
	// as authentication or logging, this is a great opportunity to do so.

	p.Configured = true
}

func (p *Provider) Resources(ctx context.Context) []func() resource.Resource {
	{{- $provider := . }}
	return []func() resource.Resource{
		{{- range $dataSource := .Resources }}
			New{{ .TfTypeNameSuffix }},
		{{- end}}
	}
}

func (p *Provider) DataSources(ctx context.Context) []func() datasource.DataSource {
	{{- $provider := . }}
	return []func() datasource.DataSource{
		{{- range $dataSource := .DataSources }}
			New{{ .TfTypeNameSuffix }}DataSource,
		{{- end}}
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &Provider{
			Version: version,
		}
	}
}
`
}

func (g *ProviderGenerator) PackageName() string {
	return g.Config.Provider.PackageName
}

func (g *ProviderGenerator) CreateTemplateData() interface{} {
	resources := make([]*config.TerraformResource, 0)
	dataSources := make([]*config.TerraformResource, 0)

	for _, res := range g.Config.Output {
		if res.TfType == config.TfTypeResource {
			resources = append(resources, res)
		} else if res.TfType == config.TfTypeDataSource {
			// Skip for now so we don't have to code generate these
			// dataSources = append(dataSources, res)
		}
	}
	return &ProviderResourceData{
		DefaultEndpoint: g.Config.Api.DefaultEndpoint,
		PackageName:     g.PackageName(),
		ProviderName:    g.Config.Provider.ProviderName(),
		Resources:       resources,
		DataSources:     dataSources,
	}
}

func (g *ProviderGenerator) Generate(destinationDirectory string) error {
	return execute(g, fmt.Sprintf("%s/provider.go", destinationDirectory))
}

func NewProviderGenerator(doc *openapi3.T, config *config.Config) *ProviderGenerator {
	return &ProviderGenerator{
		Doc:    doc,
		Config: config,
	}
}
