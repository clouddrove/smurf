package terraform

import (
	"context"
	"os"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// Import imports existing infrastructure into Terraform state.
// It allows importing resources by address and ID, with support for variables,
// variable files, custom state, and other import-specific options.
func Import(address, id, dir string, vars, varFiles []string,
	targets []string, refresh bool, state string, config string,
	allowMissing bool, useAI bool) error {

	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Setup output
	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	// Start import process
	Info("Starting Terraform import in directory: %s", dir)
	Info("Import address: %s", address)
	Info("Import ID: %s", id)

	importOptions := []tfexec.ImportOption{}

	// Handle state file
	if state != "" {
		Info("Using custom state file: %s", state)
		importOptions = append(importOptions, tfexec.State(state))
	}

	// Handle config file
	if config != "" {
		Info("Using config file: %s", config)
		importOptions = append(importOptions, tfexec.Config(config))
	}

	// Apply variables
	if len(vars) > 0 {
		for _, v := range vars {
			Info("Applying variable: %s", v)
			importOptions = append(importOptions, tfexec.Var(v))
		}
	}

	// Apply variable files
	if len(varFiles) > 0 {
		for _, vf := range varFiles {
			Info("Loading variable file: %s", vf)
			importOptions = append(importOptions, tfexec.VarFile(vf))
		}
	}

	// Note about targets - they don't apply to import commands
	if len(targets) > 0 {
		Warn("Note: --target flag is not supported for import operations and will be ignored")
	}

	// Refresh flag support - IMPORTANT: Import doesn't directly support refresh flag
	// but we can achieve similar behavior with other options if needed
	if !refresh {
		Info("Note: Refresh flag is handled automatically by Terraform import")
	}

	// Allow missing flag
	if allowMissing {
		Warn("Allow missing flag enabled - import will proceed even if configuration is incomplete")
		importOptions = append(importOptions, tfexec.AllowMissingConfig(true))
	}

	// Execute Terraform import
	err = tf.Import(context.Background(), address, id, importOptions...)
	if err != nil {
		Error("Failed to import resource: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	Success("Successfully imported resource: %s", address)
	Info("Resource with ID %s has been added to Terraform state", id)

	Info("Next steps:")
	Info("  1. Review the state: smurf stf state list | grep %s", address)
	Info("  2. Update your configuration to match the imported resource")
	Info("  3. Run 'smurf stf plan' to verify there are no differences")

	return nil
}
