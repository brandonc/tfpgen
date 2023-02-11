package main

import (
	"fmt"
	"os"

	"github.com/brandonc/tfpgen/internal/command"
	"github.com/mitchellh/cli"
)

func main() {
	c := cli.NewCLI("tfpgen", "0.1.0")
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"examine":  command.NewExamineCommand,
		"init":     command.NewInitCommand,
		"generate": command.NewGenerateCommand,
	}

	exitStatus, err := c.Run()
	if err != nil {
		fmt.Println(err)
	}

	os.Exit(exitStatus)
}
