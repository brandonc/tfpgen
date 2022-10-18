package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/brandonc/tfpgen/internal/command"
	"github.com/brandonc/tfpgen/internal/config"
	terraformJson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func removeAllUnlessDebug(t *testing.T, dir, description string) {
	t.Helper()

	t.Cleanup(func() {
		if os.Getenv("DEBUG") != "" {
			t.Logf("not deleting %s temp directory: %s", description, dir)
		} else {
			err := os.RemoveAll(dir)
			if err != nil {
				t.Log("warning: could not delete temp directory", dir)
			}
		}
	})
}

func shallowCopyAttribute(t *testing.T, att *terraformJson.SchemaAttribute) terraformJson.SchemaAttribute {
	t.Helper()

	return terraformJson.SchemaAttribute{
		Optional:        att.Optional,
		Required:        att.Required,
		Description:     att.Description,
		DescriptionKind: att.DescriptionKind,
		Sensitive:       att.Sensitive,
		AttributeType:   att.AttributeType,
		Computed:        att.Computed,
	}
}

func expectAttributesContains(t *testing.T, expected, actual map[string]*terraformJson.SchemaAttribute) {
	t.Helper()

	for key, expectedAtt := range expected {
		actualAtt, ok := actual[key]
		require.Truef(t, ok, "expected attribute not present: %s", key)

		// Shallow copy the actual/expected attribute for comparison, so that expectations don't
		// have to match the full heirarchy within.
		expectedCopy := shallowCopyAttribute(t, expectedAtt)
		actualCopy := shallowCopyAttribute(t, actualAtt)

		require.EqualValuesf(t, expectedCopy, actualCopy, "expected %#v attribute, got %#v", expectedCopy, actualCopy)

		if expectedAtt.AttributeNestedType == nil {
			continue
		}

		if actualAtt.AttributeNestedType == nil || actualAtt.AttributeNestedType.Attributes == nil {
			t.Errorf("expected attribute %s to contain nested attributes", key)
		}

		expectAttributesContains(t, expectedAtt.AttributeNestedType.Attributes, actualAtt.AttributeNestedType.Attributes)
	}
}

func setupTerraformCommand(t *testing.T, providerDir, config, commandName string, commandArgs ...string) *exec.Cmd {
	t.Helper()

	// Set up temp dir for terraform config using the built provider
	tfDir, err := os.MkdirTemp("", "tf")
	require.NoError(t, err)

	removeAllUnlessDebug(t, tfDir, "terraform")

	require.NoError(t,
		os.WriteFile(path.Join(tfDir, "test.tfrc"), []byte(fmt.Sprintf(`provider_installation {
dev_overrides {
	"brandonc/tfpgenexample" = "%s"
}
direct {}
}`, providerDir)), 0700))

	require.NoError(t,
		os.WriteFile(path.Join(tfDir, "test.tf"), []byte(config), 0700),
	)

	cmd := exec.Command(commandName, commandArgs...)
	cmd.Dir = tfDir
	cmd.Env = append(cmd.Env, fmt.Sprintf("TF_CLI_CONFIG_FILE=%s", path.Join(tfDir, "test.tfrc")))

	return cmd
}

func TestGenerate(t *testing.T) {
	cmd := command.GenerateCommand{}

	// Set up temp dir for generating and building a test provider
	tempDir, err := os.MkdirTemp("", "tfpgenexample")
	require.NoError(t, err)

	removeAllUnlessDebug(t, tempDir, "provider")

	// Establish an absolute path to the nomad-quota config fixture, overwrite
	// the relevant path to the Open API spec, and write the config to the temp dir
	openAPISpec, err := filepath.Abs("../test-fixtures/openapi3/nomad.yaml")
	require.NoError(t, err)

	config, err := config.ReadConfig("../test-fixtures/configs/nomad-quota.yaml")
	require.NoError(t, err)

	config.Filename = openAPISpec
	err = config.Write(fmt.Sprintf("%s/%s", tempDir, "tfpgen.yaml"))
	require.NoError(t, err)

	// Set the CWD to the temp dir and run the generate command
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	retVal := cmd.Run([]string{})

	if retVal != 0 {
		t.Fatalf("expected exit code 0, got %d", retVal)
	}

	t.Run("provider can build", func(t *testing.T) {
		cmd := exec.Command("go", "mod", "tidy")
		cmd.Dir = tempDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("expected no error, received %s. mod tidy output:\n\n%s", err, output)
		}

		cmd = exec.Command("go", "build", "-o", "terraform-provider-tfpgenexample")
		cmd.Dir = tempDir

		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("expected no error, received %s. Build output:\n\n%s", err, output)
		}
	})

	t.Run("test provider schema output", func(t *testing.T) {
		cmd := setupTerraformCommand(t, tempDir, `terraform {
required_providers {
	tfpgenexample = {
		source = "brandonc/tfpgenexample"
	}
}
}

resource "tfpgenexample_quota" "example" {
create_index = 0

description = "my quota"

limits = [{
	hash = "myhash1"
	region = "myregion1"
}]
}`, "terraform", "providers", "schema", "-json")

		output, err := cmd.CombinedOutput()
		require.NoError(t, err, fmt.Sprintf("unexpected error running terraform: %s", output))

		schemas := terraformJson.ProviderSchemas{}
		require.NoError(t, schemas.UnmarshalJSON(output))
		require.NoError(t, schemas.Validate())

		tfpgenSchema, ok := schemas.Schemas["registry.terraform.io/brandonc/tfpgenexample"]
		require.True(t, ok)

		quotaSchema, ok := tfpgenSchema.ResourceSchemas["tfpgenexample_quota"]
		require.True(t, ok)

		// Incomplete, but multiple nesting levels
		expectedAttr := map[string]*terraformJson.SchemaAttribute{
			"create_index": {
				AttributeType:   cty.Number,
				Optional:        true,
				DescriptionKind: "plain",
			},
			"limits": {
				Optional:        true,
				DescriptionKind: "plain",
				AttributeNestedType: &terraformJson.SchemaNestedAttributeType{
					Attributes: map[string]*terraformJson.SchemaAttribute{
						"hash": {
							AttributeType:   cty.String,
							Optional:        true,
							DescriptionKind: "plain",
						},
						"region": {
							AttributeType:   cty.String,
							Optional:        true,
							DescriptionKind: "plain",
						},
						"region_limit": {
							Optional:        true,
							DescriptionKind: "plain",
							AttributeNestedType: &terraformJson.SchemaNestedAttributeType{
								Attributes: map[string]*terraformJson.SchemaAttribute{
									"cores": {
										AttributeType:   cty.Number,
										DescriptionKind: "plain",
										Optional:        true,
									},
									"cpu": {
										AttributeType:   cty.Number,
										DescriptionKind: "plain",
										Optional:        true,
									},
									"disk_mb": {
										AttributeType:   cty.Number,
										DescriptionKind: "plain",
										Optional:        true,
									},
									"iops": {
										AttributeType:   cty.Number,
										DescriptionKind: "plain",
										Optional:        true,
									},
									"memory_max_mb": {
										AttributeType:   cty.Number,
										DescriptionKind: "plain",
										Optional:        true,
									},
									"memory_mb": {
										AttributeType:   cty.Number,
										DescriptionKind: "plain",
										Optional:        true,
									},
									"networks": {
										Optional:        true,
										DescriptionKind: "plain",
										AttributeNestedType: &terraformJson.SchemaNestedAttributeType{
											Attributes: map[string]*terraformJson.SchemaAttribute{
												"cidr": {
													Optional:        true,
													AttributeType:   cty.String,
													DescriptionKind: "plain",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		expectAttributesContains(t, expectedAttr, quotaSchema.Block.Attributes)
	})

	t.Run("provider tests can run", func(t *testing.T) {
		cmd := exec.Command("go", "test", "./...")
		cmd.Dir = tempDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("expected no error, received %s. Test output:\n\n%s", err, output)
		}
	})
}
