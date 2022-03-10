package command

import (
	"fmt"
	"strings"

	"github.com/brandonc/tfpgen/pkg/restutils"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mitchellh/cli"
)

type ExamineCommand struct{}

func (c ExamineCommand) Help() string {
	return "Usage: examine [path]\nExamine an OpenAPI spec, which will reveal an the RESTful entities tfpgen can detect before initializing a project."
}

func (c ExamineCommand) Run(args []string) int {
	doc, err := openapi3.NewLoader().LoadFromFile(args[0])

	if err != nil {
		fmt.Printf("invalid openapi3 spec: %s\n", err)
		return 2
	}

	resources := restutils.ProbeForResources(doc)

	fmt.Printf("%-32v %-64s %-16s %-16s\n", "Config Name", "Paths", "Limit", "Collection Data Source?")
	fmt.Printf("------------------------------------------------------------------------------------------------------------------------------------------\n")
	for name, resource := range resources {
		extent := ""
		if resource.IsCRUD() {
			extent = "resource"
		} else if resource.CanReadIdentity() || resource.CanReadCollection() {
			extent = "data_source"
		}

		fmt.Printf(
			"%-32s %-64v %-16s %-16v\n",
			name,
			strings.Join(resource.Paths, ", "),
			extent,
			resource.CanReadCollection() && !resource.CanReadIdentity(),
		)
	}
	return 0
}

func (c ExamineCommand) Synopsis() string {
	return "Examine an openapi 3 specification"
}

func NewExamineCommand() (cli.Command, error) {
	return ExamineCommand{}, nil
}
