package command

import (
	"fmt"
	"os"

	"github.com/brandonc/tfpgen/internal/config"
	"github.com/brandonc/tfpgen/internal/generator"
	"github.com/brandonc/tfpgen/internal/restutils"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mitchellh/cli"
)

type GenerateCommand struct {
	Config *config.Config
	Spec   *openapi3.T
}

func (c GenerateCommand) Help() string {
	return "Usage: generate\nGenerate Terraform provider code using a tfpgen configuration file."
}

func (c GenerateCommand) Run(args []string) int {
	// Ensure tfpgen.yaml exists in the cwd
	info, err := os.Stat("tfpgen.yaml")
	if err != nil || info.IsDir() {
		fmt.Printf("tfpgen.yaml not found. Run tfpgen init to create one\n")
		return 3
	}

	// Ensure tfpgen.yaml matches the config schema
	cfg, err := config.ReadConfig("tfpgen.yaml")

	if err != nil {
		fmt.Printf("invalid tfpgen.yaml: %v", err)
		return 3
	}

	// Ensure the openapi spec file defined in the config exists
	_, err = os.Stat(cfg.Filename)
	if err != nil {
		fmt.Printf("%s not found. Ensure the file path specified in tfpgen.yml is correct\n", cfg.Filename)
		return 3
	}

	// Ensure the openapi spec file can be loaded & parsed
	doc, err := openapi3.NewLoader().LoadFromFile(cfg.Filename)
	if err != nil {
		fmt.Printf("invalid openapi3 spec: %s\n", err)
		return 2
	}

	c.Config = cfg
	c.Spec = doc

	bindings, err := c.Config.AsBindings()
	if err != nil {
		fmt.Println(err)
		return 3
	}

	resources, err := restutils.BindResources(doc, bindings)
	if err != nil {
		fmt.Println(err)
		return 3
	}

	err = os.Mkdir("generated/", 0755)
	if err != nil && !os.IsExist(err) {
		fmt.Printf("could not create directory generated/: %v\n", err)
		return 3
	}

	generator.NewProviderGenerator(c.Config)

	for key := range c.Config.Output {
		resource, ok := resources[key]
		if !ok {
			fmt.Printf("could not find configured entity key \"%s\" in %s\n", key, c.Config.Filename)
			return 3
		}

		if resource.IsCRUD() {
			resourceGenerator, err := generator.NewResourceGenerator(resource, c.Spec, c.Config.Output[key])

			if err != nil {
				fmt.Printf("could not prepare resource generator: %v\n", err)
				return 3
			}

			err = resourceGenerator.Generate("provider", "generated/", "generated/")
			if err != nil {
				fmt.Printf("could not generate resource: %v\n", err)
				return 3
			}
		}
	}

	return 0
}

func (c GenerateCommand) Synopsis() string {
	return "Generate Terraform provider"
}

func NewGenerateCommand() (cli.Command, error) {
	return GenerateCommand{}, nil
}
