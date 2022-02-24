package restutils

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/brandonc/tfpgen/internal/naming"
	"github.com/getkin/kin-openapi/openapi3"
)

type ActionName string

type Emitter string

const (
	Create ActionName = "create"
	Show   ActionName = "show"
	Index  ActionName = "index"
	Update ActionName = "update"
	Delete ActionName = "delete"
)

const JsonEmitter Emitter = "json"

var successfulResponseCodes map[ActionName][]int = map[ActionName][]int{
	Create: {201, 200},
	Show:   {200, 203},
	Index:  {200, 203},
	Update: {200},
	Delete: {200, 204},
}

var wellKnownContentTypes map[string]Emitter = map[string]Emitter{
	"application/json": JsonEmitter,
}

type RESTAction struct {
	Name          ActionName
	Method        string
	OAPIOperation *openapi3.Operation
	Path          string
	OAPIPathItem  *openapi3.PathItem
}

type SpecResource struct {
	Name       string
	Paths      []string
	RESTIndex  *RESTAction
	RESTShow   *RESTAction
	RESTCreate *RESTAction
	RESTUpdate *RESTAction
	RESTDelete *RESTAction
}

type SpecAttribute struct {
	Name     string
	ReadOnly bool
	Required bool
	Schema   *openapi3.Schema
}

func (s *SpecResource) CompositeAttributes(mediaType string) []*SpecAttribute {
	attributesByName := make(CompositeAttributes)

	if s.RESTShow != nil {
		attributesByName.ExtractResponseAttributes(Show, mediaType, s.RESTShow.OAPIOperation)
	}

	if s.RESTCreate != nil {
		attributesByName.ExtractRequestAttributes(Create, mediaType, s.RESTCreate.OAPIOperation)
	}

	if s.RESTUpdate != nil {
		attributesByName.ExtractRequestAttributes(Update, mediaType, s.RESTUpdate.OAPIOperation)
	}

	attributes := make([]*SpecAttribute, 0, len(attributesByName))
	for _, attribute := range attributesByName {
		attributes = append(attributes, attribute)
	}

	return attributes
}

func ProbeForResources(doc *openapi3.T) map[string]*SpecResource {
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

		showOK, showOperation := probeBuildAction(true, resource, resource.RESTShow, Show, path, pathItem, []string{http.MethodGet})
		if showOK {
			resource.RESTShow = showOperation
		}
		deleteOK, deleteOperation := probeBuildAction(true, resource, resource.RESTDelete, Delete, path, pathItem, []string{http.MethodDelete})
		if deleteOK {
			resource.RESTDelete = deleteOperation
		}
		updateOK, updateOperation := probeBuildAction(true, resource, resource.RESTUpdate, Update, path, pathItem, []string{http.MethodPut, http.MethodPatch, http.MethodPost})
		if updateOK {
			resource.RESTUpdate = updateOperation
		}
		listOK, indexOperation := probeBuildAction(false, resource, resource.RESTIndex, Index, path, pathItem, []string{http.MethodGet})
		if listOK {
			resource.RESTIndex = indexOperation
		}
		createOK, createOperation := probeBuildAction(false, resource, resource.RESTCreate, Create, path, pathItem, []string{http.MethodPost})
		if createOK {
			resource.RESTCreate = createOperation
		}

		if showOK || deleteOK || updateOK || listOK || createOK {
			resource.Paths = append(resource.Paths, path)
		}
	}

	return result
}

func (s *SpecResource) DetermineContentMediaType() *string {
	var mediaType *string = nil

	if s.RESTShow != nil {
		mediaType, _ = probeMediaType(s.RESTShow, successfulResponseCodes[Show])
	}

	if s.RESTUpdate != nil {
		updateMediaType, err := probeMediaType(s.RESTUpdate, successfulResponseCodes[Update])
		mediaType = reconcileMediaTypes(s.Name, Update, err, mediaType, updateMediaType)
	}

	if s.RESTCreate != nil {
		createMediaType, err := probeMediaType(s.RESTCreate, successfulResponseCodes[Create])
		mediaType = reconcileMediaTypes(s.Name, Update, err, mediaType, createMediaType)
	}

	if s.RESTIndex != nil {
		listMediaType, err := probeMediaType(s.RESTIndex, successfulResponseCodes[Index])
		mediaType = reconcileMediaTypes(s.Name, Index, err, mediaType, listMediaType)
	}

	return mediaType
}

func reconcileMediaTypes(resourceKey string, action ActionName, probeError error, previousMediaType *string, newMediaType *string) *string {
	if probeError != nil && previousMediaType != nil && newMediaType != nil && previousMediaType != newMediaType {
		fmt.Printf("warning: %s %s operation response content media type does not agree with other operation(s), which are %s\n", resourceKey, action, *previousMediaType)
	} else if previousMediaType == nil && newMediaType != nil {
		return newMediaType
	}
	return previousMediaType
}

func probeMediaType(op *RESTAction, successCodes []int) (*string, error) {
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

func probeBuildAction(singleton bool, resource *SpecResource, action *RESTAction, actionName ActionName, path string, pathItem *openapi3.PathItem, probeMethods []string) (bool, *RESTAction) {
	for _, method := range probeMethods {
		oapiOp := pathItem.GetOperation(method)
		if oapiOp == nil {
			continue
		}

		if (!singleton && !strings.HasSuffix(strings.ToLower(path), "id}")) || (singleton && strings.HasSuffix(strings.ToLower(path), "id}")) {
			if action != nil {
				fmt.Printf("warning: %s already has a %s operation defined at %s\n", resource.Name, actionName, action.Path)
				return false, nil
			}

			return true, &RESTAction{
				Name:          actionName,
				Method:        method,
				OAPIPathItem:  pathItem,
				OAPIOperation: oapiOp,
				Path:          path,
			}
		}
	}
	return false, nil
}

func (r *SpecResource) IsCRUD() bool {
	return r.RESTShow != nil &&
		r.RESTUpdate != nil &&
		r.RESTDelete != nil &&
		r.RESTCreate != nil
}

func (r *SpecResource) CanReadIdentity() bool {
	return r.RESTShow != nil
}

func (r *SpecResource) CanReadCollection() bool {
	return r.RESTIndex != nil
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
