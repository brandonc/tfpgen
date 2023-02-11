package command

import (
	"fmt"

	"github.com/brandonc/tfpgen/internal/config"
	"github.com/mitchellh/cli"
)

type InitCommand struct{}

func (c InitCommand) Help() string {
	return `Usage: init [path]
Given an openapi 3 specification, generate tfpgen.yml, which allows you to configure the provider and each resource & data source.`
}

func (c InitCommand) Run(args []string) int {
	if len(args) != 1 {
		fmt.Println("missing required argument [path] specifying an OpenAPI 3 spec file")
		return 1
	}

	if err := config.InitConfig(args[0]); err != nil {
		fmt.Println(err)
		return 2
	}
	return 0
}

func (c InitCommand) Synopsis() string {
	return "Generate an initial configuration"
}

func NewInitCommand() (cli.Command, error) {
	return InitCommand{}, nil
}
