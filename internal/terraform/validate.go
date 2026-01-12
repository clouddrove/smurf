package terraform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// ValidationError represents a single validation error with details.
type ValidationError struct {
	ErrorType   string
	Description string
	Location    string
	LineNumber  int
	LineContent string
	HelpText    string
}

// CustomValidator handles validation and error formatting.
type CustomValidator struct {
	tf *tfexec.Terraform
}

// NewCustomValidator creates a new validator instance.
func NewCustomValidator(tf *tfexec.Terraform) *CustomValidator {
	return &CustomValidator{tf: tf}
}

// formatValidationError formats a Terraform-style error output.
func (cv *CustomValidator) formatValidationError(err ValidationError) string {
	var sb strings.Builder

	sb.WriteString("\n╷\n")
	sb.WriteString(fmt.Sprintf("│ %s%s\n", RedText("Error: "), err.Description))
	sb.WriteString("│\n")
	sb.WriteString(fmt.Sprintf("│   on %s line %d:\n", GreyText(err.Location), err.LineNumber))
	sb.WriteString(fmt.Sprintf("│   %d:   %s\n", err.LineNumber, err.LineContent))
	if err.HelpText != "" {
		sb.WriteString("│\n")
		sb.WriteString(fmt.Sprintf("│   %s\n", GreyText(err.HelpText)))
	}
	sb.WriteString("╵\n")

	return sb.String()
}

// ValidateWithDetails performs `terraform validate` and logs structured output.
func (cv *CustomValidator) ValidateWithDetails(ctx context.Context) error {
	if cv.tf == nil {
		return errors.New("terraform instance is nil")
	}

	Info("Starting Terraform validation...")

	valid, err := cv.tf.Validate(ctx)
	if err != nil {
		Error("Validation process failed: %v", err)
		return fmt.Errorf("validation process error: %w", err)
	}

	if valid.Valid {
		Success("Terraform configuration is valid ✅")
		return nil
	}

	Warn("Configuration is invalid (%d errors)", valid.ErrorCount)

	for _, diag := range valid.Diagnostics {
		if string(diag.Severity) == "error" {
			location := ""
			lineNumber := 0
			lineContent := ""

			if diag.Range != nil {
				location = diag.Range.Filename
				lineNumber = diag.Range.Start.Line
				lineContent = fmt.Sprintf("col %d", diag.Range.Start.Column)
			}

			validationErr := ValidationError{
				ErrorType:   diag.Summary,
				Description: diag.Detail,
				Location:    location,
				LineNumber:  lineNumber,
				LineContent: lineContent,
				HelpText:    string(diag.Severity), // casted to string
			}

			fmt.Print(cv.formatValidationError(validationErr))
		}
	}

	Error("Validation failed with %d errors", valid.ErrorCount)
	return fmt.Errorf("validation failed with %d errors", valid.ErrorCount)
}

// GetValidateTerraform initializes Terraform executor for validation.
func GetValidateTerraform(dir string) (*tfexec.Terraform, error) {
	workDir := dir
	var err error

	if workDir == "" {
		workDir, err = os.Getwd()
		if err != nil {
			Error("Failed to get working directory: %v", err)
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		Error("Terraform executable not found: %v", err)
		return nil, fmt.Errorf("terraform executable not found: %w", err)
	}

	tf, err := tfexec.NewTerraform(workDir, terraformPath)
	if err != nil {
		Error("Failed to create Terraform executor: %v", err)
		return nil, fmt.Errorf("failed to create Terraform executor: %w", err)
	}

	Info("Terraform executable initialized successfully")
	return tf, nil
}

// Validate runs terraform validate command with detailed logging.
func Validate(dir string, useAI bool) error {
	tf, err := GetValidateTerraform(dir)
	if err != nil {
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	if tf == nil {
		Error("Terraform instance is nil")
		return fmt.Errorf("terraform instance is nil")
	}

	validator := NewCustomValidator(tf)
	return validator.ValidateWithDetails(context.Background())
}
