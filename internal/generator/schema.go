package generator

import (
	"github.com/brandonc/tfpgen/internal/config"
	"github.com/brandonc/tfpgen/pkg/naming"
	"github.com/brandonc/tfpgen/pkg/restutils"
)

// TemplateFrameworkType describes the data type of an attribute
type TemplateFrameworkType struct {
	// The Terraform Plugin Framework schema type from the types package, for example, "StringType"
	// Note this field is not required when attributes contain other attributes.
	FrameworkSchemaType string

	// If the type is a list or map, this is the type of the inner element
	ElemFrameworkSchemaType string

	// The Terraform Plugin Framework attribute go data type, for example, "string"
	DataType string
}

// TemplateResourceAttribute describes a single resource attribute and can contain other nested attributes
type TemplateResourceAttribute struct {
	// A measure of attribute nesting
	NestingLevel int

	// The Terraform name to use for the attribute. See https://www.terraform.io/language/syntax/configuration#identifiers
	TfName string

	// The OpenAPI description of the property
	Description string

	// The Capital Case field name for the attribute. This casing is important because the
	// data struct that is annotated by the framework has to be publicly exposed in go code
	DataName string

	// Attribute type information
	Type TemplateFrameworkType

	// Whether or not to mark this attribute as sensitive
	Sensitive bool

	// Whether or not the attribute is required
	Required bool

	// If it's not required or computed, the attribute should be optional
	Optional bool

	// Nested attributes that belong to this attribute
	Attributes []*TemplateResourceAttribute

	// IsList determines should be true if this represents an array attribute
	IsList bool

	// IsComplex determines which type of schema this is. Complex attributes are objects and arrays.
	IsComplex bool
}

func typeOfSimple(t restutils.OASType, f restutils.OASFormat) TemplateFrameworkType {
	return TemplateFrameworkType{
		DataType: toSimpleGoType(t, f),
		FrameworkSchemaType: toSimpleSchemaType(t, f),
	}
}

func listOf(format restutils.OASFormat, elemType restutils.OASType) TemplateFrameworkType {
	return TemplateFrameworkType{
		FrameworkSchemaType: "ListType",
		ElemFrameworkSchemaType: toSimpleSchemaType(elemType, restutils.FormatNone),
		DataType: "[]"+toSimpleGoType(elemType, format),
	}
}

func toSimpleSchemaType(t restutils.OASType, f restutils.OASFormat) string {
	switch t {
	case restutils.TypeString:
		return "StringType"
	case restutils.TypeNumber, restutils.TypeInteger:
		return "NumberType"
	case restutils.TypeBoolean:
		return "BoolType"
	default:
		panic("not a simple schema type " + t.String())
	}
}

func toSimpleGoType(t restutils.OASType, f restutils.OASFormat) string {
	switch t {
	case restutils.TypeString:
		return "string"
	case restutils.TypeNumber, restutils.TypeInteger:
		switch f {
		case restutils.FormatInt64:
			return "int64"
		case restutils.FormatFloat:
			return "float32"
		case restutils.FormatDouble:
			return "float64"
		default:
			return "int"
		}
	case restutils.TypeBoolean:
		return "bool"
	default:
		panic("not a simple schmea type " + t.String())
	}
}

func templateAttribute(nestingLevel int, att *restutils.Attribute) *TemplateResourceAttribute {
	result := TemplateResourceAttribute{
		TfName:       naming.ToHCLName(att.Name),
		Description:  att.Description,
		Required:     att.Required,
		Optional:     !att.Required,
		DataName:     naming.ToTitleName(att.Name),
		NestingLevel: nestingLevel,
	}

	if att.Type.IsArrayOrObject() {
		if att.Type == restutils.TypeObject || att.Type.IsArrayOfObjects(*att.ElemType) {
			// Complex array type
			nested := make([]*TemplateResourceAttribute, 0, len(att.Attributes))
			for _, subatt := range att.Attributes {
				nested = append(nested, templateAttribute(nestingLevel+1, subatt))
			}
			result.IsComplex = true
			result.Attributes = nested
		} else if att.Type == "array" {
			// Simple array type
			result.Type = listOf(att.Format, *att.ElemType)
		}

		if att.Type == "array" {
			result.IsList = true
		}
	} else {
		result.Type = typeOfSimple(att.Type, att.Format)
	}

	result.Sensitive = att.Format == "password"

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
