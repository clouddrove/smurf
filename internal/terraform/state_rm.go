package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/clouddrove/smurf/internal/ai"
)

// StateRm removes specified resources from the Terraform state
func StateRm(dir string, addresses []string, backup bool, useAI bool) error {
	// Validate input
	if len(addresses) == 0 {
		ai.AIExplainError(useAI, "No resource addresses provided")
		return fmt.Errorf("at least one resource address must be specified")
	}

	// Initialize Terraform (to validate the directory)
	_, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Optional: List current resources before removal
	if err := listCurrentResources(dir); err != nil {
		Warn("Unable to list current resources: %v", err)
	}

	// Create backup if requested
	if backup {
		if err := createStateBackup(dir); err != nil {
			Warn("Failed to create backup: %v", err)
		}
	}

	// Remove each resource
	removed := []string{}
	failed := []string{}

	for _, addr := range addresses {
		Info("Removing resource: %s", addr)

		err := removeResource(dir, addr)
		if err != nil {
			Error("Failed to remove %s: %v", addr, err)
			failed = append(failed, addr)
		} else {
			Success("Successfully removed: %s", addr)
			removed = append(removed, addr)
		}
	}

	// Print summary
	printRmSummary(removed, failed)

	// If there were failures and AI is enabled, provide help
	if len(failed) > 0 && useAI {
		ai.AIExplainError(useAI, formatFailureMessage(failed))
	}

	return nil
}

// removeResource executes the terraform state rm command for a single resource
func removeResource(workingDir, address string) error {
	cmd := createSecureCommand(workingDir, "state", "rm", address)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("terraform state rm failed: %v", err)
	}

	return nil
}

// listCurrentResources shows current resources in state for verification
func listCurrentResources(dir string) error {
	// Use terraform state list to show current resources
	cmd := createSecureCommand(dir, "state", "list")

	output, err := cmd.Output()
	if err != nil {
		// Don't return error as this is just informational
		return nil
	}

	resources := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(resources) > 0 && resources[0] != "" {
		Info("Current resources in state (%s):", filepath.Base(dir))

		// Show first 10 resources as preview
		previewCount := 10
		if len(resources) < previewCount {
			previewCount = len(resources)
		}

		for i := 0; i < previewCount; i++ {
			if resources[i] != "" {
				fmt.Printf("  %s\n", resources[i])
			}
		}

		if len(resources) > previewCount {
			fmt.Printf("  ... and %d more\n", len(resources)-previewCount)
		}
	}

	return nil
}

// createStateBackup creates a timestamped backup of the current state file
func createStateBackup(dir string) error {
	statePath := filepath.Join(dir, "terraform.tfstate")
	backupPath := filepath.Join(dir, fmt.Sprintf("terraform.tfstate.backup.%d", time.Now().Unix()))

	// Check if state file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return fmt.Errorf("state file not found at %s", statePath)
	}

	Info("Creating backup: %s", filepath.Base(backupPath))

	// Read the state file
	input, err := os.ReadFile(statePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %v", err)
	}

	// Write to backup file
	err = os.WriteFile(backupPath, input, 0644)
	if err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	Success("Backup created: %s", filepath.Base(backupPath))
	return nil
}

// printRmSummary prints a summary of the removal operation
func printRmSummary(removed, failed []string) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	Info("Removal Summary:")

	if len(removed) > 0 {
		Success("✓ Successfully removed (%d):", len(removed))
		for _, addr := range removed {
			fmt.Printf("  - %s\n", addr)
		}
	}

	if len(failed) > 0 {
		Error("✗ Failed to remove (%d):", len(failed))
		for _, addr := range failed {
			fmt.Printf("  - %s\n", addr)
		}
	}

	fmt.Printf("\nTotal: %d removed, %d failed\n", len(removed), len(failed))
}

// formatFailureMessage creates a detailed error message for AI assistance
func formatFailureMessage(failed []string) string {
	return fmt.Sprintf("Failed to remove resources from Terraform state: %v. "+
		"Common issues include: resource not found in state, insufficient permissions, "+
		"or invalid resource address format.", failed)
}
