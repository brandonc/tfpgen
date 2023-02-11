package restutils

import (
	"net/http"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

type AttributeValues struct {
	Description string
	Type        string
	Format      string
	Required    bool
	ReadOnly    bool
}

func Test_compositeAttributes(t *testing.T) {
	t.Run("nomad", func(t *testing.T) {
		doc, err := openapi3.NewLoader().LoadFromFile("../../test-fixtures/openapi3/nomad.yaml")

		if err != nil {
			t.Fatalf("could not load nomad.yaml: %v", err)
		}

		t.Run("namespace resource", func(t *testing.T) {
			var resource = &RESTResource{
				probe: &RESTProbe{
					Document: doc,
				},
				Name:       "Namespace",
				RESTIndex:  &RESTAction{Index, http.MethodGet, "/namespace"},
				RESTCreate: &RESTAction{Create, http.MethodPost, "/namespace"},
				RESTShow:   &RESTAction{Show, http.MethodGet, "/namespace/{namespaceName}"},
				RESTUpdate: &RESTAction{Update, http.MethodPost, "/namespace/{namespaceName}"},
			}

			attributes := compositeAttributes(resource, "application/json")

			// Here are the composite attributes that should be found and some details about them.

			// namespaceName (path, required)
			// CreateIndex (body, optional)
			// Description (body, optional)
			// ModifyIndex (body, optional)
			// Name (body, optional)
			// Quota (body, optional)

			expectedAttributes := map[string]AttributeValues{
				"namespaceName": {
					Type:     "string",
					Required: true,
					ReadOnly: false,
				},
				"CreateIndex": {
					Type:     "integer",
					Required: false,
					ReadOnly: false,
				},
				"Description": {
					Type:     "string",
					Required: false,
					ReadOnly: false,
				},
				"ModifyIndex": {
					Type:     "integer",
					Required: false,
					ReadOnly: false,
				},
				"Name": {
					Type:     "string",
					Required: false,
					ReadOnly: false,
				},
				"Quota": {
					Type:     "string",
					Required: false,
					ReadOnly: false,
				},
			}

			if len(attributes) != len(expectedAttributes) {
				t.Errorf("Expected %d attributes but found %d", len(expectedAttributes), len(attributes))
			}

			for attr, c := range expectedAttributes {
				var found *Attribute = nil
				for _, search := range attributes {
					if search.Name == attr {
						found = search
						break
					}
				}

				if found == nil {
					t.Errorf("Expected an attribute named %s in compositeAttributes", attr)
					return
				}

				if found.Required != c.Required {
					t.Errorf("attribute %s Required expected %v, actual %v", attr, c.Required, found.Required)
				}

				if found.ReadOnly != c.ReadOnly {
					t.Errorf("attribute %s ReadOnly expected %v, actual %v", attr, c.ReadOnly, found.ReadOnly)
				}

				if found.Type != OASTypeFromString(c.Type) {
					t.Errorf("attribute %s Type expected %s, actual %s", attr, c.Type, found.Type)
				}

				if found.Description != c.Description {
					t.Errorf("attribute %s Description expected %s, actual %s", attr, c.Description, found.Description)
				}
			}
		})

		t.Run("quota resource", func(t *testing.T) {
			var resource = &RESTResource{
				probe: &RESTProbe{
					Document: doc,
				},
				Name:       "Quota",
				RESTIndex:  &RESTAction{Index, http.MethodGet, "/quotas"},
				RESTShow:   &RESTAction{Show, http.MethodGet, "/quota/{specName}"},
				RESTCreate: &RESTAction{Create, http.MethodPost, "/quota/{specName}"},
				RESTUpdate: &RESTAction{Update, http.MethodPost, "/quota/{specName}"},
				RESTDelete: &RESTAction{Delete, http.MethodDelete, "/quota/{specName}"},
			}

			attributes := compositeAttributes(resource, "application/json")

			expectedAttributes := map[string]AttributeValues{
				"specName": {
					Type:     "string",
					Required: true,
					ReadOnly: false,
				},
				"CreateIndex": {
					Type:     "integer",
					Required: false,
					ReadOnly: false,
				},
				"Description": {
					Type:     "string",
					Required: false,
					ReadOnly: false,
				},
				"Limits": {
					Type:     "array",
					Required: false,
					ReadOnly: false,
				},
				"ModifyIndex": {
					Type:     "integer",
					Required: false,
					ReadOnly: false,
				},
				"Name": {
					Type:     "string",
					Required: false,
					ReadOnly: false,
				},
			}

			if len(attributes) != len(expectedAttributes) {
				t.Errorf("Expected %d attributes but found %d: %v", len(expectedAttributes), len(attributes), attributes)
			}

			for attr, c := range expectedAttributes {
				var found *Attribute = nil
				for _, search := range attributes {
					if search.Name == attr {
						found = search
						break
					}
				}

				if found == nil {
					t.Errorf("Expected an attribute named %s in compositeAttributes", attr)
					return
				}

				if found.Required != c.Required {
					t.Errorf("attribute %s Required expected %v, actual %v", attr, c.Required, found.Required)
				}

				if found.ReadOnly != c.ReadOnly {
					t.Errorf("attribute %s ReadOnly expected %v, actual %v", attr, c.ReadOnly, found.ReadOnly)
				}

				if found.Type != OASTypeFromString(c.Type) {
					t.Errorf("attribute %s Type expected %s, actual %s", attr, c.Type, found.Type)
				}

				if found.Description != c.Description {
					t.Errorf("attribute %s Description expected %s, actual %s", attr, c.Description, found.Description)
				}

				expectedLimitsAttributes := map[string]AttributeValues{
					"Hash": {
						Type:     "string",
						Format:   "byte",
						Required: false,
						ReadOnly: false,
					},
					"Region": {
						Type:     "string",
						Required: false,
						ReadOnly: false,
					},
					"RegionLimit": {
						Type:     "object",
						Required: false,
						ReadOnly: false,
					},
				}

				if attr == "Limits" {
					if *found.ElemType != TypeObject {
						t.Errorf("Attribute %s ElemType expected object, actual %s", attr, found.ElemType)
					}

					if len(found.Attributes) != len(expectedLimitsAttributes) {
						t.Errorf("Expected %d Limit attributes but found %d: %v", len(expectedLimitsAttributes), len(found.Attributes), found.Attributes)
					}

					for innerAttr, c := range expectedLimitsAttributes {
						var foundLimit *Attribute = nil
						for _, search := range found.Attributes {
							if search.Name == innerAttr {
								foundLimit = search
								break
							}
						}

						if foundLimit == nil {
							t.Errorf("Expected an attribute named %s in Limits attribute", innerAttr)
							return
						}

						if foundLimit.Required != c.Required {
							t.Errorf("Attribute %s Required expected %v, actual %v", innerAttr, c.Required, foundLimit.Required)
						}

						if foundLimit.ReadOnly != c.ReadOnly {
							t.Errorf("Attribute %s ReadOnly expected %v, actual %v", innerAttr, c.ReadOnly, foundLimit.ReadOnly)
						}

						if foundLimit.Type != OASTypeFromString(c.Type) {
							t.Errorf("Attribute %s Type expected %s, actual %s", innerAttr, c.Type, foundLimit.Type)
						}

						if foundLimit.Description != c.Description {
							t.Errorf("Attribute %s Description expected %s, actual %s", innerAttr, c.Description, foundLimit.Description)
						}

						expectedRegionLimitAttributes := map[string]AttributeValues{
							"CPU": {
								Type: "integer",
							},
							"Cores": {
								Type: "integer",
							},
							"Devices": {
								Type: "array",
							},
							"DiskMB": {
								Type: "integer",
							},
							"IOPS": {
								Type: "integer",
							},
							"MemoryMB": {
								Type: "integer",
							},
							"MemoryMaxMB": {
								Type: "integer",
							},
							"Networks": {
								Type: "array",
							},
						}

						if innerAttr == "RegionLimit" {
							if len(foundLimit.Attributes) != len(expectedRegionLimitAttributes) {
								t.Errorf("Expected %d RegionLimit attributes but found %d: %v", len(expectedRegionLimitAttributes), len(foundLimit.Attributes), foundLimit.Attributes)
							}

							for innerRegionAttr, c := range expectedRegionLimitAttributes {
								var foundRegion *Attribute = nil
								for _, search := range foundLimit.Attributes {
									if search.Name == innerRegionAttr {
										foundRegion = search
										break
									}
								}

								if foundRegion == nil {
									t.Errorf("Expected an attribute named %s in RegionLimit attribute", innerAttr)
									return
								}

								if foundRegion.Required != c.Required {
									t.Errorf("Attribute %s Required expected %v, actual %v", innerAttr, c.Required, foundRegion.Required)
								}

								if foundRegion.ReadOnly != c.ReadOnly {
									t.Errorf("Attribute %s ReadOnly expected %v, actual %v", innerAttr, c.ReadOnly, foundRegion.ReadOnly)
								}

								if foundRegion.Type != OASTypeFromString(c.Type) {
									t.Errorf("Attribute %s Type expected %s, actual %s", innerAttr, c.Type, foundRegion.Type)
								}

								if foundRegion.Description != c.Description {
									t.Errorf("Attribute %s Description expected %s, actual %s", innerAttr, c.Description, foundRegion.Description)
								}

								expectedNetworkAttributes := map[string]AttributeValues{
									"CIDR": {
										Type: "string",
									},
									"DNS": {
										Type: "object",
									},
									"Device": {
										Type: "string",
									},
									"DynamicPorts": {
										Type: "array",
									},
									"IP": {
										Type: "string",
									},
									"MBits": {
										Type: "integer",
									},
									"Mode": {
										Type: "string",
									},
									"ReservedPorts": {
										Type: "array",
									},
								}

								if innerRegionAttr == "Networks" {
									if len(foundRegion.Attributes) != len(expectedNetworkAttributes) {
										t.Errorf("Expected %d Networks attributes but found %d: %v", len(expectedNetworkAttributes), len(foundRegion.Attributes), foundRegion.Attributes)
									}

									for innerNetworkAttr := range expectedNetworkAttributes {
										var foundNetwork *Attribute = nil
										for _, search := range foundRegion.Attributes {
											if search.Name == innerNetworkAttr {
												foundNetwork = search
												break
											}
										}

										if foundNetwork == nil {
											t.Errorf("Expected an attribute named %s in Networks attribute", innerNetworkAttr)
											return
										}

										expectedDNSAttributes := map[string]AttributeValues{
											"Options": {
												Type: "array",
											},
											"Searches": {
												Type: "array",
											},
											"Servers": {
												Type: "array",
											},
										}

										if innerNetworkAttr == "DNS" {
											if len(foundNetwork.Attributes) != len(expectedDNSAttributes) {
												t.Errorf("Expected %d DNS attributes but found %d: %v", len(expectedDNSAttributes), len(foundNetwork.Attributes), foundNetwork.Attributes)
											}

											for innerDNSAttr, c := range expectedDNSAttributes {
												var foundDNS *Attribute = nil
												for _, search := range foundNetwork.Attributes {
													if search.Name == innerDNSAttr {
														foundDNS = search
														break
													}
												}

												if foundDNS == nil {
													t.Errorf("Expected an attribute named %s in DNS attribute", innerDNSAttr)
													return
												}

												if foundDNS.Type != OASTypeFromString(c.Type) {
													t.Errorf("Attribute %s Type expected %s, actual %s", innerDNSAttr, c.Type, foundDNS.Type)
												}

												// All three DNS attributes are string arrays
												if *foundDNS.ElemType != TypeString {
													t.Errorf("Attribute %s Type expected %s, actual %s", innerDNSAttr, "string", foundDNS.ElemType)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		})
	})
}
