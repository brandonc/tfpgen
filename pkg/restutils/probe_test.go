package restutils

import (
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func Test_ProbeForResources(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromFile("../../test-fixtures/restlike.yaml")

	if err != nil {
		t.Fatalf("invalid fixture: %s\n", err)
	}

	probe := NewProbe(doc)
	resources := probe.ProbeForResources()

	t.Run("finds two resource", func(t *testing.T) {
		expected := 2
		actual := len(resources)

		if expected != actual {
			t.Errorf("expected %d but got %d", expected, actual)
		}
	})

	t.Run("Boards SpecResource", func(t *testing.T) {
		expected := "Boards" // Derived from path /v3/boards
		imageBoard, ok := resources[expected]

		if !ok {
			keys := make([]string, 0, len(resources))
			for k := range resources {
				keys = append(keys, k)
			}

			t.Fatalf("expected resources to contain '%s' but it contained '%s'", expected, strings.Join(keys, ", "))
		}

		t.Run("can be terraform resource", func(t *testing.T) {
			expected := true
			actual := imageBoard.IsCRUD()

			if expected != actual {
				t.Errorf("expected %v but got %v", expected, actual)
			}
		})

		t.Run("can be terraform identity datasource", func(t *testing.T) {
			expected := true
			actual := imageBoard.CanReadIdentity()

			if expected != actual {
				t.Errorf("expected %v but got %v", expected, actual)
			}
		})

		t.Run("can be terraform collection datasource", func(t *testing.T) {
			expected := true
			actual := imageBoard.CanReadCollection()

			if expected != actual {
				t.Errorf("expected %v but got %v", expected, actual)
			}
		})

		t.Run("CRUD Operations", func(t *testing.T) {
			if imageBoard.RESTCreate == nil {
				t.Error("expected \"Boards\" create action")
			}

			if imageBoard.RESTShow == nil {
				t.Error("expected \"Boards\" show action")
			}

			if imageBoard.RESTUpdate == nil {
				t.Error("expected \"Boards\" update action")
			}

			if imageBoard.RESTDelete == nil {
				t.Error("expected \"Boards\" delete action")
			}
		})

		t.Run("ProbeForAttributes", func(t *testing.T) {
			attributes := imageBoard.ProbeForAttributes("application/json")

			// board_id
			// comment_count
			// date_last_updated
			// permissions
			// assets
			// links
			// date_created
			// id
			// asset_count
			// description
			// name
			expectedLen := 11
			actualLen := len(attributes)

			if expectedLen != actualLen {
				t.Errorf("expected %v but got %v", expectedLen, actualLen)
			}
		})
	})
}
