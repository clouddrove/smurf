package terraform

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/clouddrove/smurf/internal/ai"
)

// StateReplaceProvider replaces provider in the Terraform state using 'terraform state replace-provider'
func StateReplaceProvider(dir, fromProvider, toProvider string, useAI bool) error {
	Info("Replacing provider in state:")
	Info("  From: %s", fromProvider)
	Info("  To:   %s", toProvider)

	// Ask for confirmation
	fmt.Printf("\nThis operation will replace all instances of provider %q with %q in the state.\n", fromProvider, toProvider)
	fmt.Print("Are you sure you want to proceed? (yes/no): ")

	var response string
	fmt.Scanln(&response)
	if response != "yes" {
		Warn("Operation cancelled by user")
		return nil
	}

	// Get Terraform executable path
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Build args for 'terraform state replace-provider'
	args := []string{"state", "replace-provider", fromProvider, toProvider}

	// Get the executable path
	execPath := tf.ExecPath()

	// Create and run command
	cmd := exec.CommandContext(context.Background(), execPath, args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		Error("Failed to replace provider in state: %v", err)
		Error("Output: %s", string(output))
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	Success("Successfully replaced provider %q with %q in state", fromProvider, toProvider)
	if len(output) > 0 {
		fmt.Print(string(output))
	}
	return nil
}
