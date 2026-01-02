package terraform_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	mytf "github.com/clouddrove/smurf/internal/terraform"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInit_WithMockConfig tests the Init function with a mock Terraform configuration
func TestInit_WithMockConfig(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test-terraform-init")
	require.NoError(t, err, "failed to create temp dir")
	defer os.RemoveAll(tempDir) // clean up

	tfConfig := `
terraform {
  required_version = ">= 0.14"
}

provider "null" {
  # no configuration needed
}

# A trivial resource just to have something
resource "null_resource" "example" {}
`
	mainTfPath := filepath.Join(tempDir, "main.tf")
	err = os.WriteFile(mainTfPath, []byte(tfConfig), 0600)
	require.NoError(t, err, "failed to write main.tf")

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)
	err = os.Chdir(tempDir)
	require.NoError(t, err, "failed to chdir to tempDir")

	err = mytf.Init(tempDir, true, false)
	require.NoError(t, err, "expected Init to succeed with mock config")

	terraformDir := filepath.Join(tempDir, ".terraform")
	assert.DirExists(t, terraformDir, "'.terraform' directory was not created")

	items, err := os.ReadDir(terraformDir)
	require.NoError(t, err, "failed reading contents of .terraform directory")

	for _, item := range items {
		fmt.Printf("Found in .terraform: %s\n", item.Name())
	}

	lockFilePath := filepath.Join(tempDir, ".terraform.lock.hcl")
	if _, lockErr := os.Stat(lockFilePath); lockErr == nil {
		fmt.Printf("Terraform lock file found at: %s\n", lockFilePath)
	} else {
		fmt.Println("No .terraform.lock.hcl found (this may be normal).")
	}
}
