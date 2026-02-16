package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/clouddrove/smurf/internal/ai"
)

// StatePull pulls remote state to local and outputs to stdout
func StatePull(dir string, useAI bool) error {
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	Info("Pulling remote state from directory: %s", dir)

	// StatePull EXISTS in tfexec
	state, err := tf.StatePull(context.Background())
	if err != nil {
		Error("Failed to pull remote state: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	fmt.Print(state)
	return nil
}

// StatePush pushes local state to remote backend using 'terraform state push'
func StatePush(dir string, stateFile string, force bool, useAI bool) error {
	Warn("Pushing local state to remote backend - this operation is DANGEROUS!")

	// Get Terraform executable path
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Build args for 'terraform state push'
	args := []string{"state", "push"}

	if force {
		Warn("Force flag detected - will overwrite remote state")
		args = append(args, "-force")
	}

	// Handle stdin or file
	if stateFile == "" {
		Info("Reading state from stdin")
		args = append(args, "-")
	} else {
		Info("Pushing state file: %s", stateFile)
		args = append(args, stateFile)
	}

	// Get the executable path
	execPath := tf.ExecPath()

	// Create and run command
	cmd := exec.CommandContext(context.Background(), execPath, args...)
	cmd.Dir = dir

	// If reading from stdin, pipe it
	if stateFile == "" {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			cmd.Stdin = os.Stdin
		} else {
			Error("No state file specified and no data piped to stdin")
			return fmt.Errorf("no state file or stdin data provided")
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		Error("Failed to push state: %v", err)
		Error("Output: %s", string(output))
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	Success("Successfully pushed state to remote backend")
	if len(output) > 0 {
		fmt.Print(string(output))
	}
	return nil
}
