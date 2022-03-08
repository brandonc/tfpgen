package config

import (
	"fmt"

	"github.com/brandonc/tfpgen/internal/naming"
	"github.com/brandonc/tfpgen/pkg/restutils"
	"github.com/getkin/kin-openapi/openapi3"
)

// NewTerraformResource translates a probed REST resource into a configuration entity
func NewTerraformResource(resource *restutils.SpecResource) *TerraformResource {
	mediaType := resource.DetermineContentMediaType()

	if mediaType == nil {
		fmt.Printf("warning: media type for \"%s\" could not be determined\n", resource.Name)
		return nil
	}

	if !resource.IsCRUD() && !resource.CanReadCollection() && !resource.CanReadIdentity() {
		return nil
	}

	if resource.IsCRUD() {
		return &TerraformResource{
			TfType:           TfTypeResource,
			TfTypeNameSuffix: naming.ToHCLName(resource.Name),
			MediaType:        *mediaType,
			Binding: BindingInfo{
				CreateAction: generateBinding(resource.RESTCreate),
				ReadAction:   generateBinding(resource.RESTShow),
				UpdateAction: generateBinding(resource.RESTUpdate),
				DeleteAction: generateBinding(resource.RESTDelete),
			},
		}
	}

	if resource.CanReadIdentity() {
		return &TerraformResource{
			TfType:           TfTypeDataSource,
			TfTypeNameSuffix: naming.ToHCLName(resource.Name),
			MediaType:        *mediaType,
			Binding: BindingInfo{
				ReadAction: generateBinding(resource.RESTShow),
			},
		}
	}

	return &TerraformResource{
		TfType:           TfTypeDataSource,
		TfTypeNameSuffix: naming.ToHCLName(resource.Name),
		MediaType:        *mediaType,
		Binding: BindingInfo{
			IndexAction: generateBinding(resource.RESTIndex),
		},
	}
}

func generateBinding(action *restutils.RESTAction) *ActionBinding {
	if action == nil {
		panic("cannot generate binding because this action is nil")
	}

	return &ActionBinding{
		Path:   action.Path,
		Method: action.Method,
	}
}

func defaultConfig(path string) Config {
	return Config{
		Api: ApiConfig{
			Scheme:          "bearer_token",
			DefaultEndpoint: "https://api.example.com/",
		},
		Provider: ProviderConfig{
			Name: "example",
		},
		Filename: path,
		Output:   make(map[string]*TerraformResource),
	}
}

func InitConfig(path string) error {
	doc, err := openapi3.NewLoader().LoadFromFile(path)

	if err != nil {
		return fmt.Errorf("invalid openapi3 spec: %w", err)
	}

	resources := restutils.ProbeForResources(doc)

	cfg := defaultConfig(path)

	for name, resource := range resources {
		if tfResource := NewTerraformResource(resource); tfResource != nil {
			cfg.Output[name] = tfResource
		}
	}

	if err = cfg.write("tfpgen.yaml"); err != nil {
		return err
	}

	fmt.Printf("Configuration written to tfpgen.yaml\n")
	return nil
}
