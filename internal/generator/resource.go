// work in progress & exploratory

package generator

import (
	"fmt"
	"strings"

	"github.com/brandonc/tfpgen/internal/config"
	"github.com/brandonc/tfpgen/pkg/restutils"
	"github.com/getkin/kin-openapi/openapi3"
)

// ResourceGenerator is the type that generates code for each resource
type ResourceGenerator struct {
	Doc    *openapi3.T
	Config *config.Config

	currentResource  *restutils.SpecResource
	currentTerraform *config.TerraformResource
}

// TemplateResourceData describes a single resource to be templated
type TemplateResourceData struct {
	PackageName                  string
	AcceptanceTestFunctionPrefix string
	TerraformTypeName            string
	FactoryFunctionName          string
	ConfigKey                    string
	ResourceTypeStruct           string
	ResourceStruct               string
	Description                  string
	Attributes                   []*TemplateResourceAttribute
}

// TemplateResourceAttribute describes a single resource attribute and can contain other nested attributes
type TemplateResourceAttribute struct {
	NestingLevel  int
	TfName        string
	Description   string
	FrameworkType string
	DataName      string
	SchemaType    string
	DataType      string
	Required      bool
	Optional      bool
	Attributes    []*TemplateResourceAttribute
}

var _ Generator = (*ResourceGenerator)(nil)

func toTerraformFrameworkSchemaType(tfType string) string {
	if tfType == "string" {
		return "types.StringType"
	} else if tfType == "number" {
		return "types.NumberType"
	} else if tfType == "bool" {
		return "types.BoolType"
	}
	panic(fmt.Sprintf("invalid tf type \"%s\"", tfType))
}

func toTerraformFrameworkDataType(tfType string) string {
	if tfType == "string" {
		return "types.String"
	} else if tfType == "number" {
		return "types.Number"
	} else if tfType == "bool" {
		return "types.Bool"
	}
	panic(fmt.Sprintf("invalid tf type \"%s\"", tfType))
}

func toTerraformType(specType string) string {
	if specType == "integer" {
		return "number"
	} else if specType == "string" {
		return "string"
	} else if specType == "boolean" {
		return "bool"
	}
	panic(fmt.Sprintf("invalid spec type \"%s\"", specType))
}

func indent(spaces int, v string) string {
	pad := strings.Repeat("\t", spaces)
	return pad + strings.Replace(v, "\n", "\n"+pad, -1)
}

func (g *ResourceGenerator) Template() string {
	return `// Code generated by tfpgen; DO NOT EDIT.
package {{ .PackageName }}

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type {{ .ResourceTypeStruct }} struct{}

type {{ .ResourceStruct }} struct {
	Provider Provider
}

type {{ .ResourceStruct }}Data struct {
	{{- range $attribute := .Attributes }}
	{{ .DataName }} {{ .DataType }} ` + "`tfsdk:\"{{ .TfName }}\"`" +
		`	{{- end}}
}
{{ define "SchemaAttr" }}
	"{{.TfName}}": {
		MarkdownDescription: "{{ .Description }}",
		Type:                {{ .SchemaType }},
		Required:            {{ .Required }},
		Optional:            {{ .Optional }},
	},
{{ end }}
func (t {{ .ResourceTypeStruct }}) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "TODO",
		Attributes: map[string]tfsdk.Attribute{
			{{- range $attribute := .Attributes }}{{ template "SchemaAttr" $attribute }}{{- end}}
		},
	}, nil
}

func (t {{ .ResourceTypeStruct }}) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return {{ .ResourceStruct }}{
		Provider: provider,
	}, diags
}

func (r {{ .ResourceStruct }}) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data {{ .ResourceStruct }}Data

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// example, err := d.provider.client.CreateExample(...)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
	//     return
	// }

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "created a {{ .ResourceStruct }} resource")
}

func (r {{ .ResourceStruct }}) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data {{ .ResourceStruct }}Data

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// example, err := d.provider.client.ReadExample(...)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "read a {{ .ResourceStruct }} resource")
}

func (r {{ .ResourceStruct }}) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data {{ .ResourceStruct }}Data

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// example, err := d.provider.client.UpdateExample(...)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "updated a {{ .ResourceStruct }} resource")
}

func (r {{ .ResourceStruct }}) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data {{ .ResourceStruct }}Data

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// example, err := d.provider.client.DeleteExample(...)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }

	resp.State.RemoveResource(ctx)

	tflog.Info(ctx, "deleted a {{ .ResourceStruct }} resource")
}

func (r {{ .ResourceStruct }}) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStateNotImplemented(ctx, "", resp)
	tflog.Info(ctx, "imported a {{ .ResourceStruct }} resource")
}
`
}

func (g *ResourceGenerator) PackageName() string {
	return g.Config.Provider.PackageName
}

func (g *ResourceGenerator) Generate(destinationPath string) error {
	bindings, err := g.Config.AsBindings()
	if err != nil {
		// Provided error message is adequate
		return err
	}

	resources, err := restutils.BindResources(g.Doc, bindings)
	if err != nil {
		// Provided error message is adequate
		return err
	}

	for key := range g.Config.Output {
		resource, ok := resources[key]
		if !ok {
			return fmt.Errorf("could not find configured entity key \"%s\" in %s", key, g.Config.Filename)
		}

		if resource.IsCRUD() {
			g.currentResource = resource
			g.currentTerraform = g.Config.Output[key]

			err = execute(g, fmt.Sprintf("%s/resource_%s.go", destinationPath, g.currentTerraform.TfTypeNameSuffix))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *ResourceGenerator) CreateTemplateData() interface{} {
	return &TemplateResourceData{
		PackageName:                  "provider",
		AcceptanceTestFunctionPrefix: "AccTest_",
		Attributes:                   templateAttributes(g.currentResource, g.currentTerraform),
		TerraformTypeName:            g.currentTerraform.TfTypeNameSuffix,
		FactoryFunctionName:          fmt.Sprintf("%sType_%s", g.currentTerraform.TfType, g.currentTerraform.TfTypeNameSuffix),
		ConfigKey:                    g.currentResource.Name,
		ResourceTypeStruct:           fmt.Sprintf("%sResourceType", g.currentTerraform.TfTypeNameSuffix),
		ResourceStruct:               fmt.Sprintf("Resource%s", g.currentResource.Name),
	}
}

func NewResourceGenerator(doc *openapi3.T, config *config.Config) *ResourceGenerator {
	return &ResourceGenerator{
		Doc:    doc,
		Config: config,
	}
}
