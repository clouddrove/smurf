package terraform

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
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
func (l *CustomLogger) Write(p []byte) (n int, err error) {
	msg := string(p)

	if strings.Contains(msg, "Downloading") {
		parts := strings.Split(msg, " ")
		if len(parts) >= 4 {
			pterm.Info.Printf("- Downloading %s %s for %s...\n",
				color.CyanString(parts[1]),
				color.YellowString(parts[2]),
				color.CyanString(strings.TrimSpace(parts[4])))
		}
		return len(p), nil
	}

	if strings.Contains(msg, "provider") {
		if strings.Contains(msg, "Installing") {
			parts := strings.Split(msg, " ")
			pterm.Info.Printf("- Installing %s...\n",
				color.CyanString(strings.Join(parts[1:], " ")))
		} else if strings.Contains(msg, "Reusing") {
			pterm.Info.Printf("- %s\n", msg)
		}
		return len(p), nil
	}

	if strings.Contains(msg, "Initializing") {
		section := ""
		if strings.Contains(msg, "backend") {
			section = "backend"
		} else if strings.Contains(msg, "modules") {
			section = "modules"
		} else if strings.Contains(msg, "provider") {
			section = "provider plugins"
		}
		if section != "" {
			pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).
				WithTextStyle(pterm.NewStyle(pterm.FgLightWhite)).
				Printf("Initializing %s...", section)
			fmt.Println()
		}
		return len(p), nil
	}

	if strings.Contains(msg, "successfully initialized") {
		pterm.Success.Println("\nTerraform has been successfully initialized!")
		pterm.Info.Println("\nYou may now begin working with Terraform. Try running \"smurf stf plan\" to see")
		pterm.Info.Println("any changes that are required for your infrastructure. All Terraform commands")
		pterm.Info.Println("should now work.")
		pterm.Info.Println("\nIf you ever set or change modules or backend configuration for Terraform,")
		pterm.Info.Println("rerun this command to reinitialize your working directory. If you forget, other")
		pterm.Info.Println("commands will detect it and remind you to do so if necessary.")
		return len(p), nil
	}

	return l.writer.Write(p)
}

// Init initializes the Terraform working directory by running 'init'.
// It sets up the Terraform client, executes the initialization with upgrade options,
// and provides user feedback through spinners and colored messages.
// Upon successful initialization, it configures custom writers for enhanced output.
func Init() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	logger := &CustomLogger{writer: os.Stdout}
	tf.SetStdout(logger)
	tf.SetStderr(logger)

	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).
		WithTextStyle(pterm.NewStyle(pterm.FgLightWhite)).
		Printf("Terraform Initialization")
	fmt.Println()

	spinner := pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		WithText("Running terraform init...")
	spinner.Start()

	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		spinner.Fail("Terraform initialization failed")
		pterm.Error.Printf("Error: %v\n", err)
		return err
	}

	spinner.Success("Initialization complete")
	spinner.Stop()
	return nil
}
