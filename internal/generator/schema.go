package generator

import (
	"github.com/brandonc/tfpgen/internal/config"
	"github.com/brandonc/tfpgen/pkg/naming"
	"github.com/brandonc/tfpgen/pkg/restutils"
)

func templateAttribute(nestingLevel int, att *restutils.Attribute) *TemplateResourceAttribute {
	result := TemplateResourceAttribute{
		TfName:       naming.ToHCLName(att.Name),
		Description:  att.Description,
		Required:     att.Required,
		Optional:     !att.Required,
		DataName:     naming.ToTitleName(att.Name),
		NestingLevel: nestingLevel,
	}

	if att.Type == "object" || att.Type == "array" {
		if att.Type == "object" || (att.Type == "array" && att.ElemType == "composite") {
			nested := make([]*TemplateResourceAttribute, 0, len(att.Attributes))
			for _, subatt := range att.Attributes {
				nested = append(nested, templateAttribute(nestingLevel+1, subatt))
			}
			result.IsComposite = true
			result.Attributes = nested

			if att.Type == "array" {
				result.IsList = true

				if att.ElemType == "composite" {
					result.CompositeFunction = "ListNestedAttributes"
					result.CompositeOptions = "ListNestedAttributesOptions{}"
				}
			} else {
				result.CompositeFunction = "SingleNestedAttributes"
			}
		} else if att.Type == "array" {
			// Simple array type
			result.FrameworkElemSchemaType = toTerraformFrameworkSchemaType(toTerraformType(att.ElemType))
		}
	}

	result.FrameworkSchemaType = toTerraformFrameworkSchemaType(toTerraformType(att.Type))
	result.FrameworkDataType = toTerraformFrameworkDataType(toTerraformType(att.Type))

	return &result
}

func templateAttributes(sresource *restutils.RESTResource, tresource *config.TerraformResource) []*TemplateResourceAttribute {
	attributes := sresource.ProbeForAttributes(tresource.MediaType)
	result := make([]*TemplateResourceAttribute, 0, len(attributes))

	for _, att := range attributes {
		result = append(result, templateAttribute(0, att))
	}

	return result
}
