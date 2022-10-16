package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/brandonc/tfpgen/internal/command"
	"github.com/brandonc/tfpgen/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	cmd := command.GenerateCommand{}

	// Set up temp dir for generating and building a test provider
	tempDir, err := os.MkdirTemp("", "tfpgenexample")
	require.NoError(t, err)

	t.Cleanup(func() {
		if os.Getenv("DEBUG") != "" {
			t.Log("not deleting temp directory", tempDir)
		} else {
			err := os.RemoveAll(tempDir)
			if err != nil {
				t.Log("warning: could not delete temp directory", tempDir)
			}
		}
	})

	fmt.Printf("test working directory: %s\n", tempDir)

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
