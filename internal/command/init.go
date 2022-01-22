package command

import (
	"fmt"

	"github.com/brandonc/tfpgen/internal/config"
	"github.com/mitchellh/cli"
)

type InitCommand struct{}

func (c InitCommand) Help() string {
	return "Usage: init [spec.yaml]\nGiven an openapi 3 specification, generate tfpgen.yml, which configures each resource & data source."
}

func (c InitCommand) Run(args []string) int {
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
