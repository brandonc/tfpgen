package restutils

import (
	"fmt"
	"log"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// CompositeAttributes are combined parameter, request, and response attributes for
// a particular RESTResource definition.

func isPrimitive(s *openapi3.Schema) bool {
	return s.Type == "string" || s.Type == "integer" || s.Type == "number" || s.Type == "boolean"
}

func isObject(s *openapi3.Schema) bool {
	return s.Type == "object"
}

func isArray(s *openapi3.Schema) bool {
	return s.Type == "array"
}

func isSimpleArray(s *openapi3.Schema) bool {
	return s.Type == "array" && s.Items != nil && isPrimitive(s.Items.Value)
}

func compositeAttributes(s *RESTResource, mediaType string) []*Attribute {
	attMap := make(map[string]*Attribute)

	// The path parameters and response body attributes from the show action
	// are the canonical attributes for a resource. Show actions give us the
	// full state of a resource.
	if s.RESTShow != nil {
		op := s.GetOperation(s.RESTShow)
		if op != nil {
			log.Print("[DEBUG] Extracting parameter attributes from show action")
			extractParameterAttributes(attMap, Show, op)
			log.Print("[DEBUG] Extracting response body attributes from show action")
			extractResponseAttributes(attMap, Show, mediaType, op)
		} else {
			log.Print("[WARN] No show operation found")
		}
	}

	// The request body attributes from the create action are also highly prioritized
	// due to the data requirements to create them
	if s.RESTCreate != nil {
		op := s.GetOperation(s.RESTCreate)
		if op != nil {
			log.Print("[DEBUG] Extracting parameter attributes from create action")
			extractParameterAttributes(attMap, Create, s.GetOperation(s.RESTCreate))
			log.Print("[DEBUG] Extracting request body attributes from create action")
			extractRequestAttributes(attMap, Create, mediaType, s.GetOperation(s.RESTCreate))
		} else {
			log.Print("[WARN] No create operation found")
		}
	}

	// The request body attributes from the update action are also supported
	if s.RESTUpdate != nil {
		op := s.GetOperation(s.RESTCreate)
		if op != nil {
			log.Print("[DEBUG] Extracting parameter attributes from update action")
			extractParameterAttributes(attMap, Update, s.GetOperation(s.RESTUpdate))
			log.Print("[DEBUG] Extracting request body attributes from update action")
			extractRequestAttributes(attMap, Update, mediaType, s.GetOperation(s.RESTUpdate))
		} else {
			log.Print("[WARN] No update operation found")
		}
	}

	return attributeValues(attMap)
}

// attributeValues maps an attribute map to a slice
func attributeValues(attMap map[string]*Attribute) []*Attribute {
	if attMap == nil {
		return nil
	}

	result := make([]*Attribute, 0, len(attMap))
	for _, att := range attMap {
		result = append(result, att)
	}
	return result
}

// extractParameterAttributes extracts at all openapi operation parameters that are found
// in the path. Other parameters are usually uninteresting and exhaustive for the purposes
// or resource probing.
func extractParameterAttributes(attMap map[string]*Attribute, action RESTPseudonym, op *openapi3.Operation) {
	for _, paramRef := range op.Parameters {
		parameter := paramRef.Value
		if parameter.In == "path" {
			// Implicitly required because this is a path parameter
			update(attMap, action, parameter.Name, false, true, parameter.Schema.Value)
		}
		// Other types of parameters are not substantial: cookie, header, or query
	}
}

// extractRequestAttributes recursively extracts attributes from the request body
func extractRequestAttributes(attMap map[string]*Attribute, action RESTPseudonym, mediaType string, op *openapi3.Operation) {
	if op.RequestBody != nil {
		body := op.RequestBody.Value.Content.Get(mediaType)
		if body != nil {
			extractFromSchemas(attMap, action, body.Schema.Value.Properties)
		}
	} else {
		log.Printf("[DEBUG] Action %s has no request body of type %s", action, mediaType)
	}
}

// extractRequestAttributes recursively extracts attributes from the response body
func extractResponseAttributes(attMap map[string]*Attribute, action RESTPseudonym, mediaType string, op *openapi3.Operation) {
	for _, code := range successfulResponseCodes[action] {
		if response := op.Responses.Get(code); response != nil {
			body := response.Value.Content.Get(mediaType)
			if body != nil {
				extractFromSchemas(attMap, action, body.Schema.Value.Properties)
				break
			}
		} else {
			log.Printf("[DEBUG] Action %s, code %d has no response body of type %s", action, code, mediaType)
		}
	}
}

// extractFromSchemas recursively extracts attributes from the specified OpenAPI schema,
// using the specified action to determine the attribute properties.
func extractFromSchemas(attMap map[string]*Attribute, action RESTPseudonym, schemas openapi3.Schemas) {
	for name, prop_ref := range schemas {
		if action == Index || action == Show {
			update(attMap, action, name, true, false, prop_ref.Value)
		} else if action == Create || action == Update {
			update(attMap, action, name, false, sliceIncludes(prop_ref.Value.Required, name), prop_ref.Value)
		}
	}
}

func sliceIncludes(slice []string, item string) bool {
	for _, element := range slice {
		if strings.Compare(element, item) == 0 {
			return true
		}
	}
	return false
}

func formatForLog(format string) string {
	if len(format) > 0 {
		return fmt.Sprintf(" (%s)", format)
	}
	return ""
}

// update will create or update the specified attribute map from schema, recursively extracting
// attributes from sub-schema.
func update(attMap map[string]*Attribute, action RESTPseudonym, name string, readonly bool, required bool, schema *openapi3.Schema) {
	existing, ok := attMap[name]
	if !ok {
		// This is an attribute we've not seen before.
		log.Printf("[DEBUG] Found param %s (%s) for %s", name, schema.Type, action)
		var attSub map[string]*Attribute = nil

		var elemType *OASType = nil
		if isObject(schema) && len(schema.Properties) > 0 {
			log.Printf("[DEBUG] Extracting sub-parameters for object %s", name)
			attSub = make(map[string]*Attribute)
			extractFromSchemas(attSub, action, schema.Properties)
			log.Printf("[DEBUG] ...Found %d for %s", len(attSub), name)
		} else if isArray(schema) {
			if isSimpleArray(schema) {
				e := OASTypeFromString(schema.Items.Value.Type)
				elemType = &e
			} else {
				e := TypeObject
				elemType = &e

				log.Printf("[DEBUG] Extracting sub-parameters for object array %s", name)
				attSub = make(map[string]*Attribute, 0)
				extractFromSchemas(attSub, action, schema.Items.Value.Properties)
				log.Printf("[DEBUG] ...Found %d for %s", len(attSub), name)
			}
		}

		attMap[name] = &Attribute{
			Name:        name,
			ReadOnly:    readonly,
			Type:        OASTypeFromString(schema.Type),
			ElemType:    elemType,
			Format:      OASFormatFromString(schema.Format),
			Description: schema.Description,
			Required:    required,
			Attributes:  attributeValues(attSub),
			Schema:      schema,
		}
	} else {
		// This is an attribute we've seen before. The readonly property
		// only need to be detected once per param to be set.
		if !readonly && existing.ReadOnly {
			log.Printf("[DEBUG] Param %s (%s) for %s is not read-only", name, schema.Type, action)
			setReadonlyAll(existing, false)
		}

		if string(existing.Type) != schema.Type {
			log.Printf(
				"[WARN] Expected property %s type %s%s to be %s%s",
				name, schema.Type, formatForLog(schema.Format), existing.Type, formatForLog(string(existing.Format)),
			)
		}
	}
}

// setReadonlyAll recursively sets the readonly property to true,
// indicating that the property and its subattributes are only ever
// read from the API, and not set by clients.
func setReadonlyAll(att *Attribute, value bool) {
	att.ReadOnly = value
	if att.Attributes != nil {
		for _, sub := range att.Attributes {
			setReadonlyAll(sub, value)
		}
	}
}
