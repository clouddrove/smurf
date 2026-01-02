package terraform

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// CustomLogger handles formatted output for Terraform operations
type CustomLogger struct {
	writer io.Writer
}

// Write formats the output of Terraform operations for better readability
// by the user. It formats the output for download, provider installation,
// and initialization messages.
// Write formats terraform output for readability (download, init, providers)
func (l *CustomLogger) Write(p []byte) (n int, err error) {
	msg := string(p)

	switch {
	case strings.Contains(msg, "Downloading"):
		parts := strings.Fields(msg)
		// Check for enough parts to safely access indices
		if len(parts) >= 5 { // Changed from >= 4 to >= 5 since we need parts[4]
			Info("Downloading %s %s for %s...",
				pterm.Cyan(parts[1]),
				pterm.Yellow(parts[2]),
				pterm.Cyan(strings.TrimSpace(parts[4])))
		} else {
			// Fallback to generic message if format doesn't match
			Info("Downloading: %s", strings.TrimSpace(msg))
		}
		return len(p), nil

	case strings.Contains(msg, "Installing"):
		Info("Installing provider: %s", pterm.Cyan(strings.TrimPrefix(msg, "Installing ")))
		return len(p), nil

	case strings.Contains(msg, "Reusing"):
		Info("%s", strings.TrimSpace(msg))
		return len(p), nil

	case strings.Contains(msg, "Initializing"):
		if strings.Contains(msg, "backend") {
			Info("Initializing backend...")
		} else if strings.Contains(msg, "modules") {
			Info("Initializing modules...")
		} else if strings.Contains(msg, "provider") {
			Info("Initializing provider plugins...")
		}
		return len(p), nil

	case strings.Contains(msg, "successfully initialized"):
		Success("Infrastructure successfully initialized!")
		Info("You may now begin working with Smurf. Run `smurf stf plan` to review changes.")
		return len(p), nil
	}

	// For any other output, write it as-is
	return l.writer.Write(p)
}

// Init initializes the Terraform working directory by running 'init'.
// It sets up the Terraform client, executes the initialization with upgrade options,
// and provides user feedback through spinners and colored messages.
// Upon successful initialization, it configures custom writers for enhanced output.
func Init(dir string, upgrade, useAI bool) error {
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	logger := &CustomLogger{writer: os.Stdout}
	tf.SetStdout(logger)
	tf.SetStderr(logger)

	workingDir := "."
	if dir != "" {
		workingDir = dir
	}

	Info("Starting infrastructure initialization in %s", workingDir)

	initOptions := tfexec.InitOption(
		tfexec.Upgrade(upgrade),
	)

	err = tf.Init(context.Background(), initOptions)
	if err != nil {
		Error("Error: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	Success("Terraform backend and providers successfully initialized.")
	return nil
}
