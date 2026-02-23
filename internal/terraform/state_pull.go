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

// resolveTerraformBinary resolves and validates terraform binary securely.
// This prevents PATH injection (CWE-426 / SonarQube warning).
func resolveTerraformBinary() (string, error) {
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		return "", fmt.Errorf("terraform binary not found in PATH")
	}

	info, err := os.Stat(terraformPath)
	if err != nil {
		return "", fmt.Errorf("unable to stat terraform binary: %v", err)
	}

	// Ensure binary is not world-writable
	if info.Mode().Perm()&0022 != 0 {
		return "", fmt.Errorf("terraform binary is writable; insecure PATH configuration")
	}

	return terraformPath, nil
}

// secureEnv returns restricted environment with fixed PATH.
func secureEnv() []string {
	return []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/bin",
	}
}

// StatePull fetches and displays the current remote Terraform state
func StatePull(dir string, useAI bool) error {

	// Validate terraform directory
	_, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Check backend configuration
	if err := checkBackendConfiguration(dir); err != nil {
		Warn("No remote backend configured or unable to check: %v", err)
		Info("Attempting to pull state anyway...")
	}

	Info("Pulling remote state from: %s", filepath.Base(dir))

	state, err := pullRemoteState(dir)
	if err != nil {
		Error("Failed to pull remote state: %v", err)
		ai.AIExplainError(useAI, formatPullErrorMessage(err))
		return fmt.Errorf("state pull failed: %v", err)
	}

	if prettyPrintJSON(state) != nil {
		fmt.Println(string(state))
	}

	Success("Successfully pulled remote state")
	return nil
}

// pullRemoteState executes terraform state pull securely
func pullRemoteState(workingDir string) ([]byte, error) {

	terraformPath, err := resolveTerraformBinary()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(terraformPath, "state", "pull")
	cmd.Dir = workingDir
	cmd.Env = secureEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
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

	terraformPath, err := resolveTerraformBinary()
	if err != nil {
		return err
	}

	cmd := exec.Command(terraformPath, "init", "-backend=true", "-get=false")
	cmd.Dir = dir
	cmd.Env = secureEnv()

	if cmd.Run() != nil {
		return fmt.Errorf("backend initialization check failed")
	}

	terraformDir := filepath.Join(dir, ".terraform")
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		return fmt.Errorf("terraform directory not found, run 'terraform init' first")
	}

	return nil
}

// prettyPrintJSON formats JSON with proper indentation
func prettyPrintJSON(data []byte) error {
	var prettyJSON bytes.Buffer

	err := json.Indent(&prettyJSON, data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %v", err)
	}

	_, err = prettyJSON.WriteTo(os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to write output: %v", err)
	}

	fmt.Println()
	return nil
}

// formatPullErrorMessage creates a user-friendly error message
func formatPullErrorMessage(err error) string {
	errMsg := err.Error()

	switch {
	case strings.Contains(errMsg, "no state file"):
		return "No remote state found. This could mean:\n" +
			"  • The remote backend hasn't been initialized (run 'terraform init')\n" +
			"  • No resources have been created yet\n" +
			"  • The state file doesn't exist in the remote backend"

	case strings.Contains(errMsg, "access denied") || strings.Contains(errMsg, "permission denied"):
		return "Permission denied accessing remote state. Check:\n" +
			"  • Your AWS/GCP/Azure credentials are properly configured\n" +
			"  • You have read access to the backend\n" +
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
			"  1. Run 'terraform init'\n"+
			"  2. Verify backend configuration\n"+
			"  3. Check cloud provider credentials\n"+
			"  4. Ensure network access to backend", err)
	}
}

// StatePullToFile saves state to a file securely
func StatePullToFile(dir, outputFile string, useAI bool) error {

	state, err := pullRemoteState(dir)
	if err != nil {
		Error("Failed to pull remote state: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %v", err)
	}

	err = os.WriteFile(outputFile, prettyJSON.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write state file: %v", err)
	}

	Success("Remote state saved to: %s", outputFile)
	return nil
}
