package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/brandonc/tfpgen/pkg/restutils"
	"gopkg.in/yaml.v2"
)

type TfType string
type SecurityScheme string

const (
	// TfTypeResource describes a terraform resource
	TfTypeResource   TfType = "resource"

	// TfTypeDataSource describes a terraform data source
	TfTypeDataSource TfType = "data_source"
)

const (
	// TokenSecurityScheme describes a bearer token security scheme
	TokenSecurityScheme SecurityScheme = "bearer_token"
)

// ActionBinding is the config section that describes an openAPI path/verb binding
type ActionBinding struct {
	Path   string `yaml:"path"`
	Method string `yaml:"method"`
}

// BindingInfo is the config section which binds each CRUD action to a path & verb
type BindingInfo struct {
	CreateAction *ActionBinding `yaml:"create,omitempty"`
	ReadAction   *ActionBinding `yaml:"read,omitempty"`
	UpdateAction *ActionBinding `yaml:"update,omitempty"`
	DeleteAction *ActionBinding `yaml:"delete,omitempty"`
	IndexAction  *ActionBinding `yaml:"index,omitempty"`
}

// TerraformResource is either a terraform resource or data source that can be interacted with using terraform
type TerraformResource struct {
	TfTypeNameSuffix string      `yaml:"tf_type_name_suffix"`
	TfType           TfType      `yaml:"tf_type"`
	MediaType        string      `yaml:"media_type"`
	Binding          BindingInfo `yaml:"binding"`
}

// ProviderConfig is the container for api client configuration. The provider will generate an
// API client to use with provider.
type ApiConfig struct {
	Scheme          SecurityScheme `yaml:"scheme"`
	DefaultEndpoint string    	   `yaml:"default_endpoint"`
}

// ProviderConfig is the container for provider configuration.
type ProviderConfig struct {
	// Name must be a lowercase name that identifies your provider. For example, the official
	// Active Directory provider name is "hashicorp/ad". The official Amazon Web Services provider
	// name is "hashicorp/aws" Additionally, the name you choose must also appear in the repository
	// name of the provider. For example, github.com/hashicorp/terraform-provider-ad is the
	// repository of the Active Directory provider.
	//
	// Conventionally, the resource names themselves are prefixed with the provider name so be sure
	// to add that below.
	Name string `yaml:"name"`

	// Registry is the hostname of the module registry, defaults to registry.terraform.io
	Registry string `yaml:"registry"`

	// ModuleRepository is the repository url for the go module, example: github.com/brandonc/terraform-provider-example
	ModuleRepository string `yaml:"repository"`

	// PackageName is the name of the go package to use for the provider, default is "provider"
	PackageName string `yaml:"package_name"`
}

// Config is the top level configuration schema
type Config struct {
	Api      ApiConfig                     `yaml:"api"`
	Provider ProviderConfig                `yaml:"provider"`
	Filename string                        `yaml:"specfile"`
	Output   map[string]*TerraformResource `yaml:"output"`
}

// Write writes the configuration data to the specified path
func (c *Config) Write(path string) error {
	d, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, d, 0644)
}

func ensureBinding(key string, action restutils.RESTPseudonym, ba *ActionBinding) error {
	if ba == nil || len(ba.Path) == 0 {
		return fmt.Errorf("resource %s, action %s is missing a binding", key, action)
	}
	return nil
}

func (p *ProviderConfig) ProviderNamespace() string {
	parts := strings.Split(p.Name, "/")
	if len(parts) != 2 {
		panic("invalid provider name")
	}
	return parts[0]
}

func (p *ProviderConfig) ProviderName() string {
	parts := strings.Split(p.Name, "/")
	if len(parts) != 2 {
		panic("invalid provider name")
	}
	return parts[1]
}

func (c *Config) AsBindings() ([]restutils.RESTBinding, error) {
	result := make([]restutils.RESTBinding, 0, len(c.Output))
	for key, resource := range c.Output {
		var binding restutils.RESTBinding
		var err error
		if resource.TfType == TfTypeResource {
			if err = ensureBinding(key, restutils.Create, resource.Binding.CreateAction); err != nil {
				return nil, err
			}
			if err = ensureBinding(key, restutils.Show, resource.Binding.ReadAction); err != nil {
				return nil, err
			}
			if err = ensureBinding(key, restutils.Update, resource.Binding.UpdateAction); err != nil {
				return nil, err
			}
			if err = ensureBinding(key, restutils.Delete, resource.Binding.DeleteAction); err != nil {
				return nil, err
			}
			binding = restutils.RESTBinding{
				Name: key,
				CreateAction: &restutils.ActionBinding{
					Path:   resource.Binding.CreateAction.Path,
					Method: resource.Binding.CreateAction.Method,
				},
				ReadAction: &restutils.ActionBinding{
					Path:   resource.Binding.ReadAction.Path,
					Method: resource.Binding.ReadAction.Method,
				},
				UpdateAction: &restutils.ActionBinding{
					Path:   resource.Binding.UpdateAction.Path,
					Method: resource.Binding.UpdateAction.Method,
				},
				DeleteAction: &restutils.ActionBinding{
					Path:   resource.Binding.DeleteAction.Path,
					Method: resource.Binding.DeleteAction.Method,
				},
			}
		} else if resource.TfType == TfTypeDataSource {
			if resource.Binding.ReadAction == nil && resource.Binding.IndexAction == nil {
				return nil, fmt.Errorf("resource %s is a data source but needs either a read or list binding", key)
			}
			if resource.Binding.ReadAction != nil && resource.Binding.IndexAction != nil {
				return nil, fmt.Errorf("resource %s is a data source but needs either a read or list binding (not both)", key)
			}
			if resource.Binding.ReadAction != nil {
				binding = restutils.RESTBinding{
					Name: key,
					ReadAction: &restutils.ActionBinding{
						Path:   resource.Binding.ReadAction.Path,
						Method: resource.Binding.ReadAction.Method,
					},
				}
			} else {
				binding = restutils.RESTBinding{
					Name: key,
					IndexAction: &restutils.ActionBinding{
						Path:   resource.Binding.IndexAction.Path,
						Method: resource.Binding.IndexAction.Method,
					},
				}
			}
		}
		result = append(result, binding)
	}
	return result, nil
}

func ReadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result := Config{}
	if err = yaml.Unmarshal(raw, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
