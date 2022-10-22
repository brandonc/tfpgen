package restutils

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/brandonc/tfpgen/pkg/naming"
	"github.com/getkin/kin-openapi/openapi3"
)

// RESTPseudonym is the REST endpoint pseudonym that is not necessarily
// associated with a particular HTTP method: create, show, index, update, delete
type RESTPseudonym string

const (
	// Create is the REST pseudonym for Create (Usually POST method on a collection endpoint)
	Create RESTPseudonym = "create"

	// Show is the REST pseudonym for Show (Usually GET method on a singleton endpoint, that is, an
	// endpoint that contains a unique ID)
	Show RESTPseudonym = "show"

	// Index is the REST pseudonym for List (Usually GET method on a collection endpoint)
	Index RESTPseudonym = "index"

	// Update is the REST pseudonym for Update (Usually PUT, PATCH, or POST method on a singleton endpoint)
	Update RESTPseudonym = "update"

	// Delete is the REST pseudonym for Delete (Usually DELETE or POST method on a singleton endpoint)
	Delete RESTPseudonym = "delete"
)

var successfulResponseCodes map[RESTPseudonym][]int = map[RESTPseudonym][]int{
	Create: {201, 200},
	Show:   {200, 203},
	Index:  {200, 203},
	Update: {200},
	Delete: {200, 204},
}

var wellKnownContentTypes map[string]interface{} = map[string]interface{}{
	"application/json": nil,
}

// RESTProbe is the root level type for probing OpenAPI specifications
type RESTProbe struct {
	Document *openapi3.T
}

// RESTAction is the binding between a REST pseudonym, a method, and a path.
// Several of these define all the possible CRUD actions on a conceptual
// resource and the OpenAPI "paths" that define it.
type RESTAction struct {
	Name   RESTPseudonym
	Method string
	Path   string
}

// RESTResource is the conceptual resource that defines CRUD actions.
type RESTResource struct {
	Name       string
	RESTIndex  *RESTAction
	RESTShow   *RESTAction
	RESTCreate *RESTAction
	RESTUpdate *RESTAction
	RESTDelete *RESTAction

	probe *RESTProbe
}

// Attribute is a summary of OpenAPI schema or properties that are
// helpful when translating to another definition.
type Attribute struct {
	// Name is the key name of the attribute
	Name string

	// Type is the [OpenAPI data type](https://swagger.io/specification/#data-types).
	// The possible values are integer, number, string, boolean, object, array
	Type string

	// ElemType is the OpenAPI data type of the array elements, which
	// is only set if the Type is array
	ElemType string

	// Format is the OpenAPI [data type format](https://swagger.io/specification/#data-type-format)
	Format string

	// ReadOnly indicates whether this attribute is set by create/update attributes
	// or if it is computed by the API.
	ReadOnly bool

	// Required indicates whether this attribute is required, either by a path
	// parameter or required object schema.
	Required bool

	// Description is the OpenAPI description of the attribute.
	Description string

	// Attributes are set if this is an object type or an array type with object elements.
	Attributes []*Attribute

	// Schema is a pointer to the full OpenAPI schema for the attribute
	Schema *openapi3.Schema
}

// String is a display string for the attribute
func (a *Attribute) String() string {
	return fmt.Sprintf("%s (%s)", a.Name, a.Type)
}

// NewProbe creates a new RESTProbe, specifying the OpenAPI document to probe
func NewProbe(doc *openapi3.T) RESTProbe {
	return RESTProbe{
		Document: doc,
	}
}

func (probe *RESTProbe) getOperation(path, method string) *openapi3.Operation {
	pathItem := probe.Document.Paths.Find(path)
	if pathItem != nil {
		return pathItem.GetOperation(method)
	}
	return nil
}

// Paths return the set of URL paths that are bound to all REST actions
func (s *RESTResource) Paths() []string {
	set := make(map[string]interface{})
	actions := []*RESTAction{
		s.RESTCreate, s.RESTDelete, s.RESTIndex, s.RESTShow, s.RESTUpdate,
	}

	length := 0
	for _, action := range actions {
		if action == nil {
			continue
		}
		set[action.Path] = nil
		length++
	}

	result := make([]string, 0, length)
	for path := range set {
		result = append(result, path)
	}
	return result
}

// GetOperation is a shortcut function for fetching an OpenAPI Operation
// by path and method, which normally require two nil checks
func (s *RESTResource) GetOperation(action *RESTAction) *openapi3.Operation {
	if action == nil {
		return nil
	}
	return s.probe.getOperation(action.Path, action.Method)
}

// ProbeForAttributes creates a composite view of attributes associated with
// and entire REST resource.
func (s *RESTResource) ProbeForAttributes(mediaType string) []*Attribute {
	return compositeAttributes(s, mediaType)
}

// ProbeForResources examines an openapi3 document, pairing related paths together that can
// potentially represent a CRUD resource.
func (probe *RESTProbe) ProbeForResources() map[string]*RESTResource {
	doc := probe.Document
	result := make(map[string]*RESTResource)

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
			resource = &RESTResource{
				Name:  keyName,
				probe: probe,
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
	}

	return result
}

// DetermineContentMediaType tries to resolve the shared media type used by
// a RESTResource. At this time, it only probes well known media types
// on the Show action.
func (s *RESTResource) DetermineContentMediaType() *string {
	var mediaType *string = nil

	if s.RESTShow != nil {
		mediaType, _ = s.probeMediaType(s.RESTShow, successfulResponseCodes[Show])
	}

	return mediaType
}

func (s *RESTResource) probeMediaType(op *RESTAction, successCodes []int) (*string, error) {
	for _, code := range successCodes {
		op := s.probe.getOperation(op.Path, op.Method)
		if op == nil {
			return nil, errors.New("the specified path/method was not found")
		}
		if response := op.Responses.Get(code); response != nil {
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

	return nil, errors.New("no content body types were defined on the specified operation")
}

func probeBuildAction(singleton bool, resource *RESTResource, action *RESTAction, RESTPseudonym RESTPseudonym, path string, pathItem *openapi3.PathItem, probeMethods []string) (bool, *RESTAction) {
	for _, method := range probeMethods {
		oapiOp := pathItem.GetOperation(method)
		if oapiOp == nil {
			continue
		}

		if (!singleton && !strings.HasSuffix(strings.ToLower(path), "}")) || (singleton && strings.HasSuffix(strings.ToLower(path), "}")) {
			if action != nil {
				fmt.Printf("warning: %s already has a %s operation defined at %s\n", resource.Name, RESTPseudonym, action.Path)
				return false, nil
			}

			return true, &RESTAction{
				Name:   RESTPseudonym,
				Method: method,
				Path:   path,
			}
		}
	}
	return false, nil
}

// IsCRUD describes if a resource has Show, Delete and Create actions
// which are thought to be the minimum necessary to manage it.
func (r *RESTResource) IsCRUD() bool {
	return r.RESTShow != nil &&
		r.RESTDelete != nil &&
		r.RESTCreate != nil
}

// CanUpdate describes if a resource has an Update action
func (r *RESTResource) CanUpdate() bool {
	return r.RESTUpdate != nil
}

// CanReadIdentity describes if a resource has a Show action
func (r *RESTResource) CanReadIdentity() bool {
	return r.RESTShow != nil
}

// CanReadCollection describes if a resource has an Index action
func (r *RESTResource) CanReadCollection() bool {
	return r.RESTIndex != nil
}

// makeKeyNameFromPath generates a CapitalCaseKey using the non-parameter
// parts of a path to attempt to construct a meaningful resource name.
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
