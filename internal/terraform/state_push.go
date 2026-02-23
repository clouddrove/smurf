package terraform

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/clouddrove/smurf/internal/ai"
)

// terraformCommand returns the terraform command with secure PATH
func terraformCommand() string {
	// Try to find terraform in secure locations first
	securePaths := []string{
		"/usr/bin/terraform",
		"/usr/local/bin/terraform",
		"/opt/homebrew/bin/terraform", // for macOS
	}

	for _, path := range securePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Fallback to "terraform" if not found in secure locations
	// but we'll use a sanitized PATH
	return "terraform"
}

// createSecureCommand creates an exec.Cmd with a secure environment
func createSecureCommand(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command(terraformCommand(), args...)
	cmd.Dir = dir

	// Create a clean environment with sanitized PATH
	cleanEnv := []string{
		// Only include essential, fixed paths
		"PATH=/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin",
		// Preserve other essential environment variables but filter PATH
	}

	// Copy existing environment but filter out unsafe PATH entries
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "PATH=") {
			cleanEnv = append(cleanEnv, env)
		}
	}

	cmd.Env = cleanEnv
	return cmd
}

// runTerraformCommand executes a terraform command and returns output
func runTerraformCommand(dir string, args ...string) ([]byte, error) {
	cmd := createSecureCommand(dir, args...)
	return cmd.Output()
}

// runTerraformCommandWithOutput executes a terraform command with real-time output
func runTerraformCommandWithOutput(dir string, args ...string) error {
	cmd := createSecureCommand(dir, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// StatePush pushes local state to remote backend
func StatePush(dir string, force, backup, lock bool, lockTimeout string, useAI bool) error {
	// Validate directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		Error("Directory does not exist: %s", dir)
		ai.AIExplainError(useAI, fmt.Sprintf("Directory not found: %s", dir))
		return fmt.Errorf("directory not found: %s", dir)
	}

	// Check if local state exists
	localStatePath := filepath.Join(dir, "terraform.tfstate")
	if _, err := os.Stat(localStatePath); os.IsNotExist(err) {
		Error("Local state file not found at %s", localStatePath)
		ai.AIExplainError(useAI, "Local state file not found")
		return fmt.Errorf("local state file not found")
	}

	// Get local state info for display
	localInfo, err := getBasicStateInfo(localStatePath)
	if err == nil {
		Info("Local state - Serial: %s, Resources: %s", localInfo["serial"], localInfo["resources"])
	}

	// Pull remote state for comparison
	Info("Checking remote state...")
	remoteStateData, remoteErr := runTerraformCommand(dir, "state", "pull")

	if remoteErr == nil && len(remoteStateData) > 0 {
		remoteInfo, err := getBasicStateInfoFromData(remoteStateData)
		if err == nil {
			Info("Remote state - Serial: %s, Resources: %s", remoteInfo["serial"], remoteInfo["resources"])

			// Simple comparison
			if remoteInfo["serial"] != localInfo["serial"] {
				Warn("Local and remote state serials differ")
			}
		}
	} else {
		Warn("No remote state found or unable to fetch")
		if !force {
			if !confirmAction("No remote state found. Push local state anyway?") {
				Info("Push cancelled")
				return nil
			}
		}
	}

	// Create backup if requested
	if backup && remoteErr == nil && len(remoteStateData) > 0 {
		backupPath := filepath.Join(dir, fmt.Sprintf("terraform.tfstate.backup.%d", time.Now().Unix()))
		if err := os.WriteFile(backupPath, remoteStateData, 0644); err != nil {
			Warn("Failed to create backup: %v", err)
			if !force {
				Error("Aborting push due to backup failure (use --force to override)")
				return fmt.Errorf("backup failed")
			}
		} else {
			Info("Remote state backed up to: %s", filepath.Base(backupPath))
		}
	}

	// Confirm push unless forced
	if !force {
		fmt.Println()
		if !confirmAction("Are you sure you want to push local state to remote backend?") {
			Info("Push cancelled")
			return nil
		}
	}

	// Execute state push
	Info("Pushing local state to remote backend...")

	args := []string{"state", "push"}
	if !lock {
		args = append(args, "-lock=false")
	} else if lockTimeout != "0s" {
		args = append(args, fmt.Sprintf("-lock-timeout=%s", lockTimeout))
	}
	args = append(args, "terraform.tfstate")

	if err := runTerraformCommandWithOutput(dir, args...); err != nil {
		Error("Failed to push state: %v", err)
		ai.AIExplainError(useAI, formatPushErrorMessage(err))
		return err
	}

	Success("Successfully pushed local state to remote backend")
	return nil
}

// getBasicStateInfo extracts basic info from state file
func getBasicStateInfo(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return getBasicStateInfoFromData(data)
}

// getBasicStateInfoFromData extracts basic info from state data
func getBasicStateInfoFromData(data []byte) (map[string]string, error) {
	info := make(map[string]string)

	// Very simple parsing - just extract serial and count resources
	content := string(data)

	// Extract serial (simple approach)
	serial := "0"
	if idx := strings.Index(content, `"serial":`); idx != -1 {
		endIdx := strings.Index(content[idx:], ",")
		if endIdx != -1 {
			serial = strings.TrimSpace(content[idx+8 : idx+endIdx])
		}
	}
	info["serial"] = serial

	// Count resources (simple approach)
	resourceCount := strings.Count(content, `"mode":"managed"`)
	info["resources"] = fmt.Sprintf("%d", resourceCount)

	return info, nil
}

// confirmAction prompts user for confirmation
func confirmAction(message string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n⚠️  %s [y/N] ", message)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// formatPushErrorMessage creates user-friendly error messages
func formatPushErrorMessage(err error) string {
	errMsg := err.Error()

	switch {
	case strings.Contains(errMsg, "locked"):
		return "State is locked by another process. This could mean:\n" +
			"  • Another Terraform operation is in progress\n" +
			"  • A previous operation crashed without releasing the lock\n" +
			"  • Use -lock-timeout to wait for lock or 'terraform force-unlock' to manually release"

	case strings.Contains(errMsg, "serial") && strings.Contains(errMsg, "newer"):
		return "Remote state has a higher serial number (is newer). This means:\n" +
			"  • Remote state has been modified since you last pulled\n" +
			"  • Pull the latest state first: 'terraform state pull'\n" +
			"  • Use --force only if you're sure you want to override"

	case strings.Contains(errMsg, "access denied") || strings.Contains(errMsg, "permission denied"):
		return "Permission denied. Check:\n" +
			"  • Your cloud provider credentials have write access\n" +
			"  • Backend permissions (S3 bucket policy, IAM roles, etc.)\n" +
			"  • You're authenticated with the correct account"

	default:
		return fmt.Sprintf("Failed to push state: %v\n\n"+
			"Troubleshooting steps:\n"+
			"  1. Run 'terraform init' to ensure backend is configured\n"+
			"  2. Check your cloud provider credentials\n"+
			"  3. Use 'terraform state pull' to see remote state\n"+
			"  4. Consider using --force if appropriate", err)
	}
}
