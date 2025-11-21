package terraform_test

import (
	"os"
	"path/filepath"
	"testing"

	mytf "github.com/clouddrove/smurf/internal/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApply(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "terraform-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	currentDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(currentDir)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	tfConfig := `
terraform {
    required_version = ">= 0.12"
    required_providers {
        null = {
            source  = "hashicorp/null"
            version = "~> 3.0"
        }
    }
}

provider "null" {}

resource "null_resource" "example" {
    triggers = {
        value = "example"
    }
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(tfConfig), 0644)
	require.NoError(t, err)

	tempDir := ""
	err = mytf.Init(tempDir, true)
	require.NoError(t, err)

	t.Run("apply with auto-approve", func(t *testing.T) {
		err := mytf.Apply(true, nil, nil, false, tempDir, nil, "")
		assert.NoError(t, err)
	})

	t.Run("apply without approval", func(t *testing.T) {
		oldStdin := os.Stdin
		r, w, _ := os.Pipe()
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		go func() {
			w.Write([]byte("no\n"))
			w.Close()
		}()

		err := mytf.Apply(true, nil, nil, false, tempDir, nil, "")
		assert.NoError(t, err)
	})
}
