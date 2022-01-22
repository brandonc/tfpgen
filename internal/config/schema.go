package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type TfType string

const (
	TfTypeResource   TfType = "resource"
	TfTypeDataSource TfType = "data_source"
)

type DiagnosticInfo struct {
	Paths []string `yaml:"paths,omitempty"`
}

// TerraformResource is either a terraform resource or data source that can be interacted with using terraform
type TerraformResource struct {
	TfTypeName  string          `yaml:"tf_type_name"`
	TfType      TfType          `yaml:"tf_type"`
	MediaType   string          `yaml:"media_type"`
	Diagnostics *DiagnosticInfo `yaml:"diagnostics,omitempty"`
}

// Config is the top level configuration schema
type Config struct {
	Filename string                        `yaml:"specfile"`
	Output   map[string]*TerraformResource `yaml:"output"`
}

// write writes the configuration data to the specified path
func (c *Config) write(path string) error {
	d, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, d, 0644)
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
