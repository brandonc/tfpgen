package generator

import (
	"fmt"
	"os"
	"os/exec"
	"text/template"

	"github.com/brandonc/tfpgen/internal/config"
	"github.com/getkin/kin-openapi/openapi3"
)

type GeneratorConfig struct {
	Config      *config.Config
	PackageName string
}

type Generator interface {
	PackageName() string
	Generate(destinationDirectory string) error
	CreateTemplateData() interface{}
	Template() string
}

func GenerateAll(basePath string, doc *openapi3.T, config *config.Config) error {
	moduleGenerator := NewModuleGenerator(doc, config)
	err := moduleGenerator.Generate(fmt.Sprintf("%s/", basePath))
	if err != nil {
		return fmt.Errorf("could not generate go.mod: %w", err)
	}

	mainGenerator := NewMainGenerator(doc, config)
	err = mainGenerator.Generate(fmt.Sprintf("%s/", basePath))
	if err != nil {
		return fmt.Errorf("could not generate main: %w", err)
	}

	providerGenerator := NewProviderGenerator(doc, config)
	err = providerGenerator.Generate(fmt.Sprintf("%s/provider", basePath))
	if err != nil {
		return fmt.Errorf("could not generate provider: %w", err)
	}

	resourceGenerator := NewResourceGenerator(doc, config)
	err = resourceGenerator.Generate(fmt.Sprintf("%s/provider", basePath))
	if err != nil {
		return fmt.Errorf("could not generate resources: %w", err)
	}

	fmtcmd := exec.Command("go", "fmt", "./...")
	fmtcmd.Dir = fmt.Sprintf("%s/", basePath)
	err = fmtcmd.Run()
	if err != nil {
		return fmt.Errorf("could not format generated code: %w", err)
	}

	return nil
}

func execute(generator Generator, destinationPath string) error {
	tmpl, err := template.New("").Parse(generator.Template())
	if err != nil {
		return err
	}

	data := generator.CreateTemplateData()
	f, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("cannot save to destination \"%s\": %w", destinationPath, err)
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}
