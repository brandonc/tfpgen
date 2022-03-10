package command

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/brandonc/tfpgen/internal/config"
	"github.com/brandonc/tfpgen/internal/generator"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mitchellh/cli"
)

type GenerateCommand struct {
}

func (c GenerateCommand) Help() string {
	return "Usage: generate [path]\nGenerate Terraform provider code using a tfpgen configuration file."
}

func (c GenerateCommand) Run(args []string) int {
	var basePath = "."
	if len(args) == 1 {
		basePath = args[0]
	}

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

	// Ensure the openapi spec file can be loaded & parsed
	doc, err := openapi3.NewLoader().LoadFromFile(cfg.Filename)
	if err != nil {
		fmt.Printf("invalid openapi3 spec: %s\n", err)
		return 2
	}

	// Ensure the openapi spec file defined in the config exists
	_, err = os.Stat(cfg.Filename)
	if err != nil {
		fmt.Printf("%s not found. Ensure the file path specified in tfpgen.yml is correct\n", cfg.Filename)
		return 3
	}

	dest := fmt.Sprintf("%s/provider", basePath)
	err = os.MkdirAll(dest, 0755)
	if err != nil && !os.IsExist(err) {
		fmt.Printf("could not create directory %s: %v\n", dest, err)
		return 3
	}

	err = generator.GenerateAll(basePath, doc, cfg)
	if err != nil {
		fmt.Println(err.Error())
		return 3
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = basePath
	err = cmd.Run()

	if err != nil {
		fmt.Printf("could not run `go mod tidy` in output directory. Check module dependencies and try running it again: %s\n", err.Error())
	}

	return 0
}

func (c GenerateCommand) Synopsis() string {
	return "Generate Terraform provider"
}

func NewGenerateCommand() (cli.Command, error) {
	return GenerateCommand{}, nil
}
