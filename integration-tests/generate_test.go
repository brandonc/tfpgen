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
)

func removeAllUnlessDebug(t *testing.T, dir, description string) {
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
}`, tempDir)), 0700),
		)

		require.NoError(t,
			os.WriteFile(path.Join(tfDir, "test.tf"), []byte(`terraform {
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
}
`), 0700),
		)

		cmd := exec.Command("terraform", "providers", "schema", "-json")
		cmd.Dir = tfDir
		cmd.Env = append(cmd.Env, fmt.Sprintf("TF_CLI_CONFIG_FILE=%s", path.Join(tfDir, "test.tfrc")))

		output, err := cmd.CombinedOutput()
		require.NoError(t, err, fmt.Sprintf("unexpected error running terraform: %s", output))

		schemas := terraformJson.ProviderSchemas{}
		require.NoError(t, schemas.UnmarshalJSON(output))
		require.NoError(t, schemas.Validate())

		tfpgenSchema, ok := schemas.Schemas["registry.terraform.io/brandonc/tfpgenexample"]
		require.True(t, ok)

		quotaSchema, ok := tfpgenSchema.ResourceSchemas["tfpgenexample_quota"]
		require.True(t, ok)

		createIndexAttr, ok := quotaSchema.Block.Attributes["create_index"]
		require.True(t, ok)
		require.True(t, createIndexAttr.Optional)
	})

	t.Run("provider tests can run", func(t *testing.T) {
		t.Skip()

		cmd := exec.Command("go", "test", "./...")
		cmd.Dir = tempDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("expected no error, received %s. Test output:\n\n%s", err, output)
		}
	})
}
