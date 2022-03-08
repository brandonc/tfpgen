package config

import (
	"testing"

	"github.com/brandonc/tfpgen/pkg/restutils"
	"github.com/getkin/kin-openapi/openapi3"
)

func Test_NewTerraformResource(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromFile("../../test-fixtures/restlike.yaml")

	if err != nil {
		t.Fatalf("invalid fixture: %s\n", err)
	}

	resources := restutils.ProbeForResources(doc)

	t.Run("Boards resource", func(t *testing.T) {
		boards, ok := resources["Boards"]
		if !ok {
			t.Fatal("Expected \"Boards\" resource")
		}

		tfr := NewTerraformResource(boards)

		if tfr == nil {
			t.Fatal("Expected terraform resource")
		}

		if tfr.TfType != "resource" {
			t.Error("Expected resource terraform resource type")
		}

		if tfr.MediaType != "application/json" {
			t.Error("Expected application/json media type")
		}

		if tfr.TfTypeNameSuffix != "boards" {
			t.Error("Expected name suffix to be \"boards\"")
		}

		if tfr.Binding.CreateAction == nil {
			t.Error("Expected boards CreateAction to not be nil")
		}

		if tfr.Binding.ReadAction == nil {
			t.Error("Expected boards ReadAction to not be nil")
		}

		if tfr.Binding.UpdateAction == nil {
			t.Error("Expected boards UpdateAction to not be nil")
		}

		if tfr.Binding.DeleteAction == nil {
			t.Error("Expected boards DeleteAction to not be nil")
		}
	})
}
