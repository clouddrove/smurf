package terraform

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/clouddrove/smurf/internal/ai"
)

// StateRm removes items from the Terraform state using 'terraform state rm'
func StateRm(dir string, addresses []string, backupPath string, useAI bool) error {
	Info("Removing %d resource(s) from state", len(addresses))
	for _, addr := range addresses {
		Info("  - %s", addr)
	}

	// Get Terraform executable path
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Build args for 'terraform state rm'
	args := []string{"state", "rm"}

	if backupPath != "" {
		Info("Creating state backup at: %s", backupPath)
		args = append(args, "-backup="+backupPath)
	}

	args = append(args, addresses...)

	// Get the executable path
	execPath := tf.ExecPath()

	// Create and run command
	cmd := exec.CommandContext(context.Background(), execPath, args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		Error("Failed to remove state item(s): %v", err)
		Error("Output: %s", string(output))
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	Success("Successfully removed %d resource(s) from state", len(addresses))
	if len(output) > 0 {
		fmt.Print(string(output))
	}
	return nil
}
