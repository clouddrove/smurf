package terraform

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/clouddrove/smurf/internal/ai"
)

// StateMv moves or renames items in the Terraform state using 'terraform state mv'
func StateMv(dir, source, destination string, backupPath string, useAI bool) error {
	Info("Moving state item from '%s' to '%s'", source, destination)

	// Get Terraform executable path
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Build args for 'terraform state mv'
	args := []string{"state", "mv"}

	if backupPath != "" {
		Info("Creating state backup at: %s", backupPath)
		args = append(args, "-backup="+backupPath)
	}

	args = append(args, source, destination)

	// Get the executable path
	execPath := tf.ExecPath()

	// Create and run command
	cmd := exec.CommandContext(context.Background(), execPath, args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		Error("Failed to move state item: %v", err)
		Error("Output: %s", string(output))
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	Success("Successfully moved state item from '%s' to '%s'", source, destination)
	if len(output) > 0 {
		fmt.Print(string(output))
	}
	return nil
}
