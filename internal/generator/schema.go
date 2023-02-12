package generator

import (
	"github.com/brandonc/tfpgen/internal/config"
	"github.com/brandonc/tfpgen/pkg/naming"
	"github.com/brandonc/tfpgen/pkg/restutils"
)

type FrameworkAttributeSchemaString string
type FrameworkTypeString string

// These constants refer to variables in package github.com/hashicorp/terraform-plugin-framework/resource/schema
const (
	SchemaBool         FrameworkAttributeSchemaString = "BoolAttribute"
	SchemaFloat64      FrameworkAttributeSchemaString = "Float64Attribute"
	SchemaInt64        FrameworkAttributeSchemaString = "Int64Attribute"
	SchemaList         FrameworkAttributeSchemaString = "ListAttribute"
	SchemaMap          FrameworkAttributeSchemaString = "MapAttribute"
	SchemaNumber       FrameworkAttributeSchemaString = "NumberAttribute"
	SchemaObject       FrameworkAttributeSchemaString = "ObjectAttribute"
	SchemaSet          FrameworkAttributeSchemaString = "SetAttribute"
	SchemaString       FrameworkAttributeSchemaString = "StringAttribute"
	SchemaListNested   FrameworkAttributeSchemaString = "ListNestedAttribute"
	SchemaMapNested    FrameworkAttributeSchemaString = "MapNestedAttribute"
	SchemaSetNested    FrameworkAttributeSchemaString = "SetNestedAttribute"
	SchemaSingleNested FrameworkAttributeSchemaString = "SingleNestedAttribute"
)

// These constants refer to variables in package github.com/hashicorp/terraform-plugin-framework/types
const (
	// TypeBool is a string reference to types.BoolType
	TypeBool FrameworkTypeString = "BoolType"
	// TypeFloat64 is a string reference to types.Float64Type
	TypeFloat64 FrameworkTypeString = "Float64Type"
	// TypeInt64 is a string reference to types.Int64Type
	TypeInt64 FrameworkTypeString = "Int64Type"
	// TypeNumber is a string reference to types.NumberType
	TypeNumber FrameworkTypeString = "NumberType"
	// TypeString is a string reference to types.StringType
	TypeString FrameworkTypeString = "StringType"
)

// TemplateFrameworkType describes the data type of an attribute
type TemplateResourceAttributeSchema struct {
	// The Terraform Plugin Framework schema attribute from the resource schema package, for example, "StringAttribute"
	// Note this field is not required when attributes contain other attributes.
	FrameworkSchemaAttributeType FrameworkAttributeSchemaString

	// If the attribute is a list or map, this is the type of the inner element
	ElementType FrameworkTypeString

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
	Schema TemplateResourceAttributeSchema

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

func typeOfSimple(t restutils.OASType, f restutils.OASFormat) TemplateResourceAttributeSchema {
	return TemplateResourceAttributeSchema{
		DataType:                     toSimpleGoType(t, f),
		FrameworkSchemaAttributeType: toSimpleFrameworkSchemaType(t, f),
	}
}

func listOf(format restutils.OASFormat, elemType restutils.OASType) TemplateResourceAttributeSchema {
	return TemplateResourceAttributeSchema{
		FrameworkSchemaAttributeType: SchemaList,
		ElementType:                  toSimpleFrameworkType(elemType, restutils.FormatNone),
		DataType:                     "[]" + toSimpleGoType(elemType, format),
	}
}

func toSimpleFrameworkType(t restutils.OASType, f restutils.OASFormat) FrameworkTypeString {
	switch t {
	case restutils.TypeString:
		return TypeString
	case restutils.TypeNumber, restutils.TypeInteger:
		return TypeNumber
	case restutils.TypeBoolean:
		return TypeBool
	default:
		panic("not a simple schema type " + t.String())
	}
}

func toSimpleFrameworkSchemaType(t restutils.OASType, f restutils.OASFormat) FrameworkAttributeSchemaString {
	switch t {
	case restutils.TypeString:
		return SchemaString
	case restutils.TypeNumber, restutils.TypeInteger:
		return SchemaNumber
	case restutils.TypeBoolean:
		return SchemaBool
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
		panic("not a simple schema type " + t.String())
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
			for _, subAtt := range att.Attributes {
				nested = append(nested, templateAttribute(nestingLevel+1, subAtt))
			}
			result.IsComplex = true
			result.Attributes = nested
		} else if att.Type == "array" {
			// Simple array type
			result.Schema = listOf(att.Format, *att.ElemType)
		}

		if att.Type == "array" {
			result.IsList = true
		}
	} else {
		result.Schema = typeOfSimple(att.Type, att.Format)
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
