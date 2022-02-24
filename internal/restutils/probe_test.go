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

	resources := ProbeForResources(doc)

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

		t.Run("CompositeAttributes", func(t *testing.T) {
			attributes := imageBoard.CompositeAttributes("application/json")

			expectedLen := 10 // name and description
			actualLen := len(attributes)

			if expectedLen != actualLen {
				t.Errorf("expected %v but got %v", expectedLen, actualLen)
			}

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
			for _, att := range attributes {
				if att.Name == "comment_count" {
					if !att.ReadOnly || att.Schema.Type != "integer" {
						t.Errorf("attribute %s had unexpected properties", att.Name)
					}
					continue
				}
				if att.Name == "date_last_updated" {
					if !att.ReadOnly || att.Schema.Type != "string" {
						t.Errorf("attribute %s had unexpected properties", att.Name)
					}
					continue
				}
				if att.Name == "permissions" {
					if !att.ReadOnly || att.Schema.Type != "object" {
						t.Errorf("attribute %s had unexpected properties", att.Name)
					}
					continue
				}
				if att.Name == "assets" {
					if !att.ReadOnly || att.Schema.Type != "array" {
						t.Errorf("attribute %s had unexpected properties", att.Name)
					}
					continue
				}
				if att.Name == "links" {
					if !att.ReadOnly || att.Schema.Type != "object" {
						t.Errorf("attribute %s had unexpected properties", att.Name)
					}
					continue
				}
				if att.Name == "date_created" {
					if !att.ReadOnly || att.Schema.Type != "string" {
						t.Errorf("attribute %s had unexpected properties", att.Name)
					}
					continue
				}
				if att.Name == "id" {
					if !att.ReadOnly || att.Schema.Type != "string" {
						t.Errorf("attribute %s had unexpected properties", att.Name)
					}
					continue
				}
				if att.Name == "asset_count" {
					if !att.ReadOnly || att.Schema.Type != "integer" {
						t.Errorf("attribute %s had unexpected properties", att.Name)
					}
					continue
				}
				if att.Name == "name" {
					if att.ReadOnly || att.Schema.Type != "string" {
						t.Errorf("attribute %s had unexpected properties", att.Name)
					}
					continue
				}
				if att.Name == "description" {
					if att.ReadOnly || att.Schema.Type != "string" {
						t.Errorf("attribute %s had unexpected properties", att.Name)
					}
					continue
				}
			}
		})
	})
}
