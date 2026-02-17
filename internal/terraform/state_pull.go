package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/clouddrove/smurf/internal/ai"
)

// StatePull fetches and displays the current remote Terraform state
func StatePull(dir string, useAI bool) error {
	// Initialize Terraform (to validate the directory)
	_, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Check if backend is configured
	if err := checkBackendConfiguration(dir); err != nil {
		Warn("No remote backend configured or unable to check: %v", err)
		Info("Attempting to pull state anyway...")
	}

	// Pull the remote state
	Info("Pulling remote state from: %s", filepath.Base(dir))

	state, err := pullRemoteState(dir)
	if err != nil {
		Error("Failed to pull remote state: %v", err)
		ai.AIExplainError(useAI, formatPullErrorMessage(err))
		return fmt.Errorf("state pull failed: %v", err)
	}

	// Pretty print the JSON
	if err := prettyPrintJSON(state); err != nil {
		// If pretty printing fails, just output raw
		fmt.Println(string(state))
	}

	Success("Successfully pulled remote state")
	return nil
}

// pullRemoteState executes terraform state pull command
func pullRemoteState(workingDir string) ([]byte, error) {
	cmd := exec.Command("terraform", "state", "pull")
	cmd.Dir = workingDir

	// Capture stdout and stderr separately
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Include stderr in error message for debugging
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%v: %s", err, stderr.String())
		}
		return nil, err
	}

	if stdout.Len() == 0 {
		return nil, fmt.Errorf("received empty state")
	}

	return stdout.Bytes(), nil
}

// checkBackendConfiguration verifies if a remote backend is configured
func checkBackendConfiguration(dir string) error {
	// Run terraform init to ensure backend is initialized
	cmd := exec.Command("terraform", "init", "-backend=true", "-get=false")
	cmd.Dir = dir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("backend initialization check failed")
	}

	// Check if backend is configured by looking at .terraform directory
	terraformDir := filepath.Join(dir, ".terraform")
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		return fmt.Errorf("terraform directory not found, run 'terraform init' first")
	}

	return nil
}

// prettyPrintJSON formats JSON with proper indentation
func prettyPrintJSON(data []byte) error {
	var prettyJSON bytes.Buffer

	// Try to parse and prettify
	err := json.Indent(&prettyJSON, data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %v", err)
	}

	// Write to stdout
	_, err = prettyJSON.WriteTo(os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to write output: %v", err)
	}

	fmt.Println() // Add newline at the end
	return nil
}

// formatPullErrorMessage creates a user-friendly error message
func formatPullErrorMessage(err error) string {
	errMsg := err.Error()

	// Common error patterns
	switch {
	case strings.Contains(errMsg, "no state file"):
		return "No remote state found. This could mean:\n" +
			"  • The remote backend hasn't been initialized (run 'terraform init')\n" +
			"  • No resources have been created yet\n" +
			"  • The state file doesn't exist in the remote backend"

	case strings.Contains(errMsg, "access denied") || strings.Contains(errMsg, "permission denied"):
		return "Permission denied accessing remote state. Check:\n" +
			"  • Your AWS/GCP/Azure credentials are properly configured\n" +
			"  • You have read access to the backend (S3 bucket, GCS bucket, etc.)\n" +
			"  • The backend configuration is correct"

	case strings.Contains(errMsg, "context deadline exceeded") || strings.Contains(errMsg, "timeout"):
		return "Timeout while connecting to remote backend. Check:\n" +
			"  • Your network connection\n" +
			"  • The backend endpoint is accessible\n" +
			"  • Proxy/firewall settings"

	case strings.Contains(errMsg, "no such host"):
		return "Cannot resolve backend hostname. Check:\n" +
			"  • Your DNS configuration\n" +
			"  • The backend endpoint URL is correct"

	default:
		return fmt.Sprintf("Failed to pull remote state: %v\n\n"+
			"Troubleshooting steps:\n"+
			"  1. Run 'terraform init' to initialize the backend\n"+
			"  2. Verify your backend configuration in terraform files\n"+
			"  3. Check your cloud provider credentials\n"+
			"  4. Ensure you have network access to the backend", err)
	}
}

// StatePullToFile is an additional helper function to save state to a file
func StatePullToFile(dir, outputFile string, useAI bool) error {
	state, err := pullRemoteState(dir)
	if err != nil {
		Error("Failed to pull remote state: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Format JSON
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %v", err)
	}

	// Write to file
	err = os.WriteFile(outputFile, prettyJSON.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write state file: %v", err)
	}

	Success("Remote state saved to: %s", outputFile)
	return nil
}
