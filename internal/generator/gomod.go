package generator

import (
	"fmt"

	"github.com/brandonc/tfpgen/internal/config"
	"github.com/getkin/kin-openapi/openapi3"
)

type ModuleGenerator struct {
	Config *config.Config
	Doc    *openapi3.T
}

type ModuleGeneratorData struct {
	Repository string
}

var _ Generator = (*ModuleGenerator)(nil)

func (g *ModuleGenerator) Template() string {
	return `module {{ .Repository }}

go 1.19
`
}

func (g *ModuleGenerator) CreateTemplateData() interface{} {
	return &ModuleGeneratorData{
		Repository: g.Config.Provider.ModuleRepository,
	}
}

func (g *ModuleGenerator) PackageName() string {
	return ""
}

func (g *ModuleGenerator) Generate(destinationDirectory string) error {
	return execute(g, fmt.Sprintf("%s/go.mod", destinationDirectory))
}

func NewModuleGenerator(doc *openapi3.T, config *config.Config) *ModuleGenerator {
	return &ModuleGenerator{
		Config: config,
		Doc:    doc,
	}
}
