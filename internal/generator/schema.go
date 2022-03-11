package generator

import (
	"fmt"
	"strings"

	"github.com/brandonc/tfpgen/internal/config"
	"github.com/brandonc/tfpgen/pkg/naming"
	"github.com/brandonc/tfpgen/pkg/restutils"
	"github.com/getkin/kin-openapi/openapi3"
)

func sliceIncludes(slice []string, item string) bool {
	for _, element := range slice {
		if strings.EqualFold(element, item) {
			return true
		}
	}
	return false
}

func schemaToAttribute(nestingLevel int, name string, required bool, schema *openapi3.Schema) *TemplateResourceAttribute {
	templateAttribute := TemplateResourceAttribute{
		TfName:       naming.ToHCLName(name),
		Description:  "TODO",
		Required:     required,
		Optional:     !required,
		DataName:     naming.ToTitleName(name),
		NestingLevel: nestingLevel,
	}

	// TODO: Does "additionalProperties" suggest a map?

	if schema.Type == "object" {
		nested := make([]*TemplateResourceAttribute, 0, len(schema.Properties))
		for propName, propRef := range schema.Properties {
			prop := propRef.Value
			nested = append(nested, schemaToAttribute(nestingLevel+1, propName, sliceIncludes(prop.Required, propName), prop))
		}
		templateAttribute.Attributes = nested
		templateAttribute.FrameworkSchemaType = "types.ObjectType"
		templateAttribute.FrameworkDataType = "types.Object"
	} else if schema.Type == "array" {
		prop := schema.Items.Value
		if prop.Type != "object" {
			// This can be represented as a list
			templateAttribute.FrameworkSchemaType = "types.ListType"
			templateAttribute.FrameworkDataType = "types.List"
		} else {
			// TODO: ListNestedAttributes
			fmt.Printf("warning: ListNestedAttributes not yet supported for property %s\n", name)
		}
	} else {
		templateAttribute.FrameworkSchemaType = toTerraformFrameworkSchemaType(toTerraformType(schema.Type))
		templateAttribute.FrameworkDataType = toTerraformFrameworkDataType(toTerraformType(schema.Type))
	}

	return &templateAttribute
}

// Extract request and response parameters into Terraform schema:
// 1. Required request parameters become required Terraform attributes.
// 2. Other request parameters become optional Terraform attributes.
// 3. Response attributes that do not appear in the request become Computed attributes.
func templateAttributes(sresource *restutils.SpecResource, tresource *config.TerraformResource) []*TemplateResourceAttribute {
	specAttributes := sresource.CompositeAttributes(tresource.MediaType)
	result := make([]*TemplateResourceAttribute, 0, len(specAttributes))

	for _, att := range specAttributes {
		result = append(result, schemaToAttribute(0, att.Name, att.ReadOnly, att.Schema))
	}

	return result
}
