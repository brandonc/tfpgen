package restutils

import (
	"fmt"
	"log"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

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

	if s.RESTShow != nil {
		op := s.Operation(s.RESTShow)
		if op != nil {
			log.Print("[DEBUG] Extracting parameter attributes from show action")
			extractParameterAttributes(attMap, Show, op)
			log.Print("[DEBUG] Extracting response body attributes from show action")
			extractResponseAttributes(attMap, Show, mediaType, op)
		} else {
			log.Print("[WARN] No show operation found")
		}
	}

	if s.RESTCreate != nil {
		op := s.Operation(s.RESTCreate)
		if op != nil {
			log.Print("[DEBUG] Extracting parameter attributes from create action")
			extractParameterAttributes(attMap, Create, s.Operation(s.RESTCreate))
			log.Print("[DEBUG] Extracting request body attributes from create action")
			extractRequestAttributes(attMap, Create, mediaType, s.Operation(s.RESTCreate))
		} else {
			log.Print("[WARN] No create operation found")
		}
	}

	if s.RESTUpdate != nil {
		op := s.Operation(s.RESTCreate)
		if op != nil {
			log.Print("[DEBUG] Extracting parameter attributes from update action")
			extractParameterAttributes(attMap, Update, s.Operation(s.RESTUpdate))
			log.Print("[DEBUG] Extracting request body attributes from update action")
			extractRequestAttributes(attMap, Update, mediaType, s.Operation(s.RESTUpdate))
		} else {
			log.Print("[WARN] No update operation found")
		}
	}

	return attributeValues(attMap)
}

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

func extractParameterAttributes(attMap map[string]*Attribute, action ActionName, op *openapi3.Operation) {
	for _, paramRef := range op.Parameters {
		parameter := paramRef.Value
		if parameter.In == "path" {
			// Implicitly required because this is a path parameter
			update(attMap, action, parameter.Name, false, true, parameter.Schema.Value)
		}
		// Other types of parameters are not substantial: cookie, header, or query
	}
}

func extractRequestAttributes(attMap map[string]*Attribute, action ActionName, mediaType string, op *openapi3.Operation) {
	if op.RequestBody != nil {
		body := op.RequestBody.Value.Content.Get(mediaType)
		if body != nil {
			extractFromSchemas(attMap, action, body.Schema.Value.Properties)
		}
	} else {
		log.Printf("[DEBUG] Action %s has no request body of type %s", action, mediaType)
	}
}

func extractResponseAttributes(attMap map[string]*Attribute, action ActionName, mediaType string, op *openapi3.Operation) {
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

func sliceIncludes(slice []string, item string) bool {
	for _, element := range slice {
		if strings.Compare(element, item) == 0 {
			return true
		}
	}
	return false
}

func extractFromSchemas(attMap map[string]*Attribute, action ActionName, schemas openapi3.Schemas) {
	for name, prop_ref := range schemas {
		if action == Index || action == Show {
			update(attMap, action, name, true, false, prop_ref.Value)
		} else if action == Create || action == Update {
			update(attMap, action, name, false, sliceIncludes(prop_ref.Value.Required, name), prop_ref.Value)
		}
	}
}

func formatForLog(format string) string {
	if len(format) > 0 {
		return fmt.Sprintf(" (%s)", format)
	}
	return ""
}

func update(attMap map[string]*Attribute, action ActionName, name string, readonly bool, required bool, schema *openapi3.Schema) {
	existing, ok := attMap[name]
	if !ok {
		log.Printf("[DEBUG] Found param %s (%s) for %s", name, schema.Type, action)
		var attSub map[string]*Attribute = nil

		elemType := ""
		if isObject(schema) && len(schema.Properties) > 0 {
			log.Printf("[DEBUG] Extracting sub-parameters for object %s", name)
			attSub = make(map[string]*Attribute)
			extractFromSchemas(attSub, action, schema.Properties)
			log.Printf("[DEBUG] ...Found %d for %s", len(attSub), name)
		} else if isArray(schema) {
			if isSimpleArray(schema) {
				elemType = schema.Items.Value.Type
			} else {
				elemType = "composite"

				log.Printf("[DEBUG] Extracting sub-parameters for composite array %s", name)
				attSub = make(map[string]*Attribute, 0)
				extractFromSchemas(attSub, action, schema.Items.Value.Properties)
				log.Printf("[DEBUG] ...Found %d for %s", len(attSub), name)
			}
		}

		attMap[name] = &Attribute{
			Name:        name,
			ReadOnly:    readonly,
			Type:        schema.Type,
			ElemType:    elemType,
			Format:      schema.Format,
			Description: schema.Description,
			Required:    required,
			Attributes:  attributeValues(attSub),
		}
	} else {
		// readonly and required only need to be detected once per param to be set
		// So it would not be advisable to update these values every time update is called
		if !readonly && existing.ReadOnly {
			log.Printf("[DEBUG] Param %s (%s) for %s is not read-only", name, schema.Type, action)
			setReadonlyAll(existing, false)
		}
		if existing.Type != schema.Type {
			log.Printf(
				"[WARN] Expected property %s type %s%s to be %s%s",
				name, schema.Type, formatForLog(schema.Format), existing.Type, formatForLog(existing.Format),
			)
		}
	}
}

func setReadonlyAll(att *Attribute, value bool) {
	att.ReadOnly = value
	if att.Attributes != nil {
		for _, sub := range att.Attributes {
			setReadonlyAll(sub, value)
		}
	}
}
