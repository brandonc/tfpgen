package config

import (
	"fmt"

	"github.com/brandonc/tfpgen/internal/naming"
	"github.com/brandonc/tfpgen/internal/specutils"
	"github.com/getkin/kin-openapi/openapi3"
)

// newTerraformResource translates a probed REST resource into a configuration entity
func newTerraformResource(resource *specutils.SpecResource) *TerraformResource {
	var terraformType TfType = TfTypeResource
	mediaType := resource.DetermineContentMediaType()

	if mediaType == nil {
		fmt.Printf("warning: media type for \"%s\" could not be determined\n", resource.Name)
		return nil
	}

	if !resource.IsCRUD() && (resource.CanReadCollection() || resource.CanReadIdentity()) {
		terraformType = TfTypeDataSource
	} else if !resource.IsCRUD() {
		return nil
	}

	return &TerraformResource{
		TfType:     terraformType,
		TfTypeName: naming.ToHCLName(resource.Name),
		MediaType:  *mediaType,
		Diagnostics: &DiagnosticInfo{
			Paths: resource.Paths,
		},
	}
}

func InitConfig(path string) error {
	doc, err := openapi3.NewLoader().LoadFromFile(path)

	if err != nil {
		return fmt.Errorf("invalid openapi3 spec: %w", err)
	}

	resources := specutils.ProbeForRESTResources(doc)

	cfg := Config{
		Filename: path,
		Output:   make(map[string]*TerraformResource),
	}

	for name, resource := range resources {
		if tfresource := newTerraformResource(resource); tfresource != nil {
			cfg.Output[name] = tfresource
		}
	}

	if err = cfg.write("tfpgen.yaml"); err != nil {
		return err
	}

	fmt.Printf("Configuration written to tfpgen.yaml\n")
	return nil
}
