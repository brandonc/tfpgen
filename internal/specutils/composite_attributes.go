package specutils

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

type CompositeAttributes map[string]*SpecAttribute

func (c CompositeAttributes) ExtractRequestAttributes(action Action, mediaType string, op *openapi3.Operation) {
	if op.RequestBody != nil {
		body := op.RequestBody.Value.Content.Get(mediaType)
		if body != nil {
			c.extractFromSchemas(action, body.Schema.Value.Properties)
		}
	}
}

func (c CompositeAttributes) ExtractResponseAttributes(action Action, mediaType string, op *openapi3.Operation) {
	for _, code := range successfulResponseCodes[action] {
		if response := op.Responses.Get(code); response != nil {
			body := response.Value.Content.Get(mediaType)
			if body != nil {
				c.extractFromSchemas(action, body.Schema.Value.Properties)
				break
			}
		}
	}
}

func (c CompositeAttributes) extractFromSchemas(action Action, schemas openapi3.Schemas) {
	for name, prop_ref := range schemas {
		if action == List || action == Show {
			c.updateForRead(name, prop_ref.Value)
		} else if action == Create || action == Update {
			c.updateForWrite(name, prop_ref.Value)
		}
	}
}

func (c CompositeAttributes) updateForRead(name string, schema *openapi3.Schema) {
	existing, ok := c[name]
	if !ok {
		c[name] = &SpecAttribute{
			Name:     name,
			ReadOnly: true,
			Schema:   schema,
		}
	} else {
		if existing.Schema.Type != schema.Type {
			fmt.Printf("warning: expected property %s type %s to be %s\n", name, schema.Type, existing.Schema.Type)
		}
	}
}

func (c CompositeAttributes) updateForWrite(name string, schema *openapi3.Schema) {
	existing, ok := c[name]
	if !ok {
		c[name] = &SpecAttribute{
			Name:     name,
			ReadOnly: false,
			Schema:   schema,
		}
	} else {
		existing.ReadOnly = false

		if existing.Schema.Type != schema.Type {
			fmt.Printf("warning: expected property %s type %s to be %s\n", name, schema.Type, existing.Schema.Type)
		}
	}
}
