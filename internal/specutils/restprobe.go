package specutils

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/brandonc/tfpgen/internal/naming"
	"github.com/getkin/kin-openapi/openapi3"
)

type Action string

type Emitter string

const (
	Create Action = "create"
	Show   Action = "show"
	List   Action = "list"
	Update Action = "update"
	Delete Action = "delete"
)

const JsonEmitter Emitter = "json"

var successfulResponseCodes map[Action][]int = map[Action][]int{
	Create: {201, 200},
	Show:   {200, 203},
	List:   {200, 203},
	Update: {200},
	Delete: {200, 204},
}

var wellKnownContentTypes map[string]Emitter = map[string]Emitter{
	"application/json": JsonEmitter,
}

type Operation struct {
	Action        Action
	OAPIOperation *openapi3.Operation
	Path          string
	OAPIPathItem  *openapi3.PathItem
}

type SpecResource struct {
	Name            string
	Paths           []string
	ListOperation   *Operation
	ShowOperation   *Operation
	CreateOperation *Operation
	UpdateOperation *Operation
	DeleteOperation *Operation
}

type SpecAttribute struct {
	Name     string
	ReadOnly bool
	Required bool
	Schema   *openapi3.Schema
}

func (s *SpecResource) CompositeAttributes(mediaType string) []*SpecAttribute {
	attributesByName := make(CompositeAttributes)

	if s.ShowOperation != nil {
		attributesByName.ExtractResponseAttributes(Show, mediaType, s.ShowOperation.OAPIOperation)
	}

	if s.CreateOperation != nil {
		attributesByName.ExtractRequestAttributes(Create, mediaType, s.CreateOperation.OAPIOperation)
	}

	if s.UpdateOperation != nil {
		attributesByName.ExtractRequestAttributes(Update, mediaType, s.UpdateOperation.OAPIOperation)
	}

	attributes := make([]*SpecAttribute, 0, len(attributesByName))
	for _, attribute := range attributesByName {
		attributes = append(attributes, attribute)
	}

	return attributes
}

func ProbeForRESTResources(doc *openapi3.T) map[string]*SpecResource {
	result := make(map[string]*SpecResource)

	paths := make([]string, 0, len(doc.Paths))
	for k := range doc.Paths {
		paths = append(paths, k)
	}

	prefix := naming.FindPrefix(paths)

	for path, pathItem := range doc.Paths {
		keyName := makeKeyNameFromPath(path[len(prefix):])
		if keyName == "" {
			keyName = makeKeyNameFromPath(path)
		}

		resource, ok := result[keyName]
		if !ok {
			resource = &SpecResource{
				Name:  keyName,
				Paths: make([]string, 0, 1),
			}

			result[keyName] = resource
		}

		// Each path can have multiple actions assigned to it, a composite of which could be used as a RESTful set.

		showOK, showOperation := maybeBuildOperation(true, resource, resource.ShowOperation, Show, path, pathItem, []*openapi3.Operation{pathItem.Get})
		if showOK {
			resource.ShowOperation = showOperation
		}
		deleteOK, deleteOperation := maybeBuildOperation(true, resource, resource.DeleteOperation, Delete, path, pathItem, []*openapi3.Operation{pathItem.Delete, pathItem.Post, pathItem.Put, pathItem.Patch})
		if deleteOK {
			resource.DeleteOperation = deleteOperation
		}
		updateOK, updateOperation := maybeBuildOperation(true, resource, resource.UpdateOperation, Update, path, pathItem, []*openapi3.Operation{pathItem.Put, pathItem.Patch, pathItem.Post})
		if updateOK {
			resource.UpdateOperation = updateOperation
		}
		listOK, listOperation := maybeBuildOperation(false, resource, resource.ListOperation, List, path, pathItem, []*openapi3.Operation{pathItem.Get})
		if listOK {
			resource.ListOperation = listOperation
		}
		createOK, createOperation := maybeBuildOperation(false, resource, resource.CreateOperation, Create, path, pathItem, []*openapi3.Operation{pathItem.Post})
		if createOK {
			resource.CreateOperation = createOperation
		}

		if showOK || deleteOK || updateOK || listOK || createOK {
			resource.Paths = append(resource.Paths, path)
		}
	}

	return result
}

func (s *SpecResource) DetermineContentMediaType() *string {
	var mediaType *string = nil

	if s.ShowOperation != nil {
		mediaType, _ = probeMediaType(s.ShowOperation, successfulResponseCodes[Show])
	}

	if s.UpdateOperation != nil {
		updateMediaType, err := probeMediaType(s.UpdateOperation, successfulResponseCodes[Update])
		mediaType = reconcileMediaTypes(s.Name, Update, err, mediaType, updateMediaType)
	}

	if s.CreateOperation != nil {
		createMediaType, err := probeMediaType(s.CreateOperation, successfulResponseCodes[Create])
		mediaType = reconcileMediaTypes(s.Name, Update, err, mediaType, createMediaType)
	}

	if s.ListOperation != nil {
		listMediaType, err := probeMediaType(s.ListOperation, successfulResponseCodes[List])
		mediaType = reconcileMediaTypes(s.Name, List, err, mediaType, listMediaType)
	}

	return mediaType
}

func reconcileMediaTypes(resourceKey string, action Action, probeError error, previousMediaType *string, newMediaType *string) *string {
	if probeError != nil && previousMediaType != nil && newMediaType != nil && previousMediaType != newMediaType {
		fmt.Printf("warning: %s %s operation response content media type does not agree with other operation(s), which are %s\n", resourceKey, action, *previousMediaType)
	} else if previousMediaType == nil && newMediaType != nil {
		return newMediaType
	}
	return previousMediaType
}

func probeMediaType(op *Operation, successCodes []int) (*string, error) {
	for _, code := range successCodes {
		if response := op.OAPIOperation.Responses.Get(code); response != nil {
			keys := make([]string, 0, len(response.Value.Content))
			for k := range response.Value.Content {
				keys = append(keys, k)

				if _, ok := wellKnownContentTypes[k]; ok {
					return &k, nil
				}
			}

			// Just return the first content type found
			if len(keys) > 0 {
				return &keys[0], nil
			}
		}
	}

	return nil, fmt.Errorf("no content body types were defined on the specified operation")
}

func maybeBuildOperation(singleton bool, resource *SpecResource, resourceOp *Operation, action Action, path string, pathItem *openapi3.PathItem, oapiOps []*openapi3.Operation) (bool, *Operation) {
	for _, oapiOp := range oapiOps {
		if oapiOp != nil {
			if (!singleton && !strings.HasSuffix(strings.ToLower(path), "id}")) || (singleton && strings.HasSuffix(strings.ToLower(path), "id}")) {
				if resourceOp != nil {
					fmt.Printf("warning: %s already has a %s operation defined at %s\n", resource.Name, action, resourceOp.Path)
					return false, nil
				}

				return true, &Operation{
					Action:        action,
					OAPIPathItem:  pathItem,
					OAPIOperation: oapiOp,
					Path:          path,
				}
			}
		}
	}
	return false, nil
}

func (r *SpecResource) IsCRUD() bool {
	return r.ShowOperation != nil &&
		r.UpdateOperation != nil &&
		r.DeleteOperation != nil &&
		r.CreateOperation != nil
}

func (r *SpecResource) CanReadIdentity() bool {
	return r.ShowOperation != nil
}

func (r *SpecResource) CanReadCollection() bool {
	return r.ListOperation != nil
}

func makeKeyNameFromPath(path string) string {
	parts := strings.Split(path, "/")

	result := ""
	for _, part := range parts {
		if strings.HasPrefix(part, "{") {
			continue
		}
		s := make([]rune, 0, len(part))
		capitalizeNext := true
		for _, c := range part {
			if !unicode.IsDigit(c) && !unicode.IsLetter(c) {
				capitalizeNext = true
				continue
			}

			if capitalizeNext {
				c = unicode.ToUpper(c)
				capitalizeNext = false
			}
			s = append(s, c)
		}
		result += string(s)
	}
	return result
}
