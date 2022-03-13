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

type RESTProbe struct {
	Document *openapi3.T
}

type RESTAction struct {
	Name   ActionName
	Method string
	Path   string
}

type RESTResource struct {
	Name       string
	RESTIndex  *RESTAction
	RESTShow   *RESTAction
	RESTCreate *RESTAction
	RESTUpdate *RESTAction
	RESTDelete *RESTAction

	probe *RESTProbe
}

type Attribute struct {
	Name        string
	Type        string
	ElemType    string
	Format      string
	ReadOnly    bool
	Required    bool
	Description string
	Attributes  []*Attribute
}

func (a *Attribute) String() string {
	return fmt.Sprintf("%s (%s)", a.Name, a.Type)
}

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

func (s *RESTResource) Paths() []string {
	set := make(map[string]interface{})
	actions := []*RESTAction{
		s.RESTCreate, s.RESTDelete, s.RESTIndex, s.RESTShow, s.RESTUpdate,
	}

	length := 0
	for _, action := range actions {
		set[action.Path] = nil
		length++
	}

	result := make([]string, 0, length)
	for path, _ := range set {
		result = append(result, path)
	}
	return result
}

func (s *RESTResource) Operation(action *RESTAction) *openapi3.Operation {
	if action == nil {
		return nil
	}
	return s.probe.getOperation(action.Path, action.Method)
}

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

func probeBuildAction(singleton bool, resource *RESTResource, action *RESTAction, actionName ActionName, path string, pathItem *openapi3.PathItem, probeMethods []string) (bool, *RESTAction) {
	for _, method := range probeMethods {
		oapiOp := pathItem.GetOperation(method)
		if oapiOp == nil {
			continue
		}

		if (!singleton && !strings.HasSuffix(strings.ToLower(path), "}")) || (singleton && strings.HasSuffix(strings.ToLower(path), "}")) {
			if action != nil {
				fmt.Printf("warning: %s already has a %s operation defined at %s\n", resource.Name, actionName, action.Path)
				return false, nil
			}

			return true, &RESTAction{
				Name:   actionName,
				Method: method,
				Path:   path,
			}
		}
	}
	return false, nil
}

func (r *RESTResource) IsCRUD() bool {
	return r.RESTShow != nil &&
		r.RESTDelete != nil &&
		r.RESTCreate != nil
}

func (r *RESTResource) CanUpdate() bool {
	return r.RESTUpdate != nil
}

func (r *RESTResource) CanReadIdentity() bool {
	return r.RESTShow != nil
}

func (r *RESTResource) CanReadCollection() bool {
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
