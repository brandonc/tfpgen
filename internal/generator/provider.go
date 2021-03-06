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
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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

func (p *Provider) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"endpoint": {
				MarkdownDescription: "The HTTP API endpoint for the provider",
				Optional:            true,
				Type:                types.StringType,
			},
			"api_token": {
				MarkdownDescription: "The HTTP API token, sent as Authorization: Bearer header",
				Optional:            true,
				Type:                types.StringType,
			},
		},
	}, nil
}

func (p *Provider) Configure(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
	var data providerData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Endpoint.Null {
		data.Endpoint.Value = "{{ .DefaultEndpoint }}"
	}

	// If the upstream provider SDK or HTTP client requires configuration, such
	// as authentication or logging, this is a great opportunity to do so.

	p.Configured = true
}

func (p *Provider) GetResources(ctx context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	{{- $provider := . }}
	return map[string]tfsdk.ResourceType{
		{{- range $resource := .Resources }}
			"{{ $provider.ProviderName }}_{{ .TfTypeNameSuffix }}": {{ .TfTypeNameSuffix }}ResourceType{},
		{{- end}}
	}, nil
}

func (p *Provider) GetDataSources(ctx context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	{{- $provider := . }}
	return map[string]tfsdk.DataSourceType{
		{{- range $dataSource := .DataSources }}
			"{{ $provider.ProviderName }}_{{ .TfTypeNameSuffix }}": {{ .TfTypeNameSuffix }}DataSourceType{},
		{{- end}}
	}, nil
}

func New(version string) func() tfsdk.Provider {
	return func() tfsdk.Provider {
		return &Provider{
			Version: version,
		}
	}
}

// convertProviderType is a helper function for NewResource and NewDataSource
// implementations to associate the concrete provider type. Alternatively,
// this helper can be skipped and the provider type can be directly type
// asserted (e.g. provider: in.(*provider)), however using this can prevent
// potential panics.
func convertProviderType(in tfsdk.Provider) (Provider, diag.Diagnostics) {
	var diags diag.Diagnostics

	p, ok := in.(*Provider)

	if !ok {
		diags.AddError(
			"Unexpected Provider Instance Type",
			fmt.Sprintf("While creating the data source or resource, an unexpected provider type (%T) was received. This is always a bug in the provider code and should be reported to the provider developers.", p),
		)
		return Provider{}, diags
	}

	if p == nil {
		diags.AddError(
			"Unexpected Provider Instance Type",
			"While creating the data source or resource, an unexpected empty provider instance was received. This is always a bug in the provider code and should be reported to the provider developers.",
		)
		return Provider{}, diags
	}

	return *p, diags
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
