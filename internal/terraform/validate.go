package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// ValidationError represents a single validation error with its details
type ValidationError struct {
	ErrorType   string
	Description string
	Location    string
	LineNumber  int
	LineContent string
	HelpText    string
}

// CustomValidator handles the validation process and error formatting
type CustomValidator struct {
	tf *tfexec.Terraform
}

// NewCustomValidator creates a new validator instance
func NewCustomValidator(tf *tfexec.Terraform) *CustomValidator {
	return &CustomValidator{
		tf: tf,
	}
}

// formatValidationError formats a single validation error in Terraform style
func (cv *CustomValidator) formatValidationError(err ValidationError) string {
	var sb strings.Builder

	errorSymbol := color.New(color.FgRed).Sprint("│")
	errorPrefix := color.New(color.FgRed).Sprint("Error: ")
	locationColor := color.New(color.FgWhite).Sprint(err.Location)
	lineNumColor := color.New(color.FgWhite).Sprint(fmt.Sprintf("line %d", err.LineNumber))

	sb.WriteString("╷\n")
	sb.WriteString(fmt.Sprintf("%s %s%s\n", errorSymbol, errorPrefix, err.Description))
	sb.WriteString(fmt.Sprintf("%s\n", errorSymbol))
	sb.WriteString(fmt.Sprintf("%s   on %s %s:\n", errorSymbol, locationColor, lineNumColor))

	sb.WriteString(fmt.Sprintf("%s   %d:   %s\n", errorSymbol, err.LineNumber, err.LineContent))

	if err.HelpText != "" {
		sb.WriteString(fmt.Sprintf("%s\n", errorSymbol))
		sb.WriteString(fmt.Sprintf("%s %s\n", errorSymbol, err.HelpText))
	}

	sb.WriteString("╵\n")
	return sb.String()
}

// ValidateWithDetails performs validation and returns detailed error output
func (cv *CustomValidator) ValidateWithDetails(ctx context.Context) error {
	if cv.tf == nil {
		return fmt.Errorf("Terraform instance is nil")
	}
	spinner, _ := pterm.DefaultSpinner.Start("Validating Terraform configuration...")

	valid, err := cv.tf.Validate(ctx)
	if err != nil {
		spinner.Fail("Validation process failed")
		return fmt.Errorf("validation process error: %w", err)
	}

	if valid.Valid {
		spinner.Success("Terrafform Configuration is valid")
		return nil
	}

	spinner.Fail(fmt.Sprintf("Configuration is invalid (%d errors)", valid.ErrorCount))

	for _, diag := range valid.Diagnostics {
		if diag.Severity == "error" {
			validationErr := ValidationError{
				ErrorType:   diag.Summary,
				Description: diag.Detail,
				Location:    diag.Range.Filename,
				LineNumber:  diag.Range.Start.Line,
				LineContent: fmt.Sprintf("%d", diag.Range.Start.Column),
				HelpText:    string(diag.Severity),
			}

			fmt.Print(cv.formatValidationError(validationErr))
		}
	}

	return fmt.Errorf("validation failed with %d errors", valid.ErrorCount)
}

// Helper function to extract line content from file
func getLineContent(filename string, lineNum int) (string, error) {
	return "", nil
}

// GetTerraform initializes and returns a Terraform instance
func GetValidateTerraform() (*tfexec.Terraform, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, fmt.Errorf("terraform executable not found: %w", err)
	}

	tf, err := tfexec.NewTerraform(workDir, terraformPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Terraform executor: %w", err)
	}

	return tf, nil
}

func Validate() error {
	tf, err := GetValidateTerraform()
	if err != nil {
		return err
	}

	if tf == nil {
		return fmt.Errorf("Terraform instance is nil")
	}

	validator := NewCustomValidator(tf)
	return validator.ValidateWithDetails(context.Background())
}
