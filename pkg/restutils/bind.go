package restutils

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// In describes where (in the protocol) an attribute is bound
type In string

const (
	// InPath describes that the attribute is bound in the path
	InPath    In = "path"
	InContent In = "content"
	InQuery   In = "query"
	InCookie  In = "cookie"
	InHeader  In = "header"
)

type ActionBinding struct {
	Path   string
	Method string
}

type AttributeBinding struct {
	In         In
	Name       string
	SpecType   string
	Attributes []*AttributeBinding
}

type RESTBinding struct {
	Name                string
	CreateAction        *ActionBinding
	ReadAction          *ActionBinding
	UpdateAction        *ActionBinding
	DeleteAction        *ActionBinding
	IndexAction         *ActionBinding
	CompositeAttributes []*AttributeBinding
}

func (p *RESTProbe) BindResources(bindings []RESTBinding) (map[string]*RESTResource, error) {
	result := make(map[string]*RESTResource)

	for _, binding := range bindings {
		var createOp, showOp, updateOp, deleteOp, listOp *RESTAction = nil, nil, nil, nil, nil
		var err error

		if binding.CreateAction != nil {
			createOp, err = bindOperation(p.Document, binding.CreateAction, Create)
			if err != nil {
				return nil, err
			}
		}
		if binding.ReadAction != nil {
			showOp, err = bindOperation(p.Document, binding.ReadAction, Show)
			if err != nil {
				return nil, err
			}
		}
		if binding.UpdateAction != nil {
			updateOp, err = bindOperation(p.Document, binding.UpdateAction, Update)
			if err != nil {
				return nil, err
			}
		}
		if binding.DeleteAction != nil {
			deleteOp, err = bindOperation(p.Document, binding.DeleteAction, Delete)
			if err != nil {
				return nil, err
			}
		}
		if binding.IndexAction != nil {
			listOp, err = bindOperation(p.Document, binding.IndexAction, Index)
			if err != nil {
				return nil, err
			}
		}

		result[binding.Name] = &RESTResource{
			probe:      p,
			Name:       binding.Name,
			RESTCreate: createOp,
			RESTShow:   showOp,
			RESTUpdate: updateOp,
			RESTDelete: deleteOp,
			RESTIndex:  listOp,
		}
	}

	return result, nil
}

func bindOperation(doc *openapi3.T, binding *ActionBinding, action RESTPseudonym) (*RESTAction, error) {
	pathItem, ok := doc.Paths[binding.Path]

	if !ok {
		return nil, fmt.Errorf("cannot bind %s to %s: path not found", action, binding.Path)
	}
	operation := pathItem.GetOperation(binding.Method)

	if operation == nil {
		return nil, fmt.Errorf("cannot bind %s to %s: operation not found", action, binding.Path)
	}

	return &RESTAction{
		Name:   action,
		Method: binding.Method,
		Path:   binding.Path,
	}, nil
}
