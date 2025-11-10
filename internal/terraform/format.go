package terraform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
)

// FormatError represents a single formatting error with details
type FormatError struct {
	ErrorType   string
	Description string
	Location    string
	LineNumber  int
	LineContent string
	NextLine    string
	HelpText    string
}

// CustomFormatter handles the formatting process and error logging
type CustomFormatter struct {
	tf      *tfexec.Terraform
	workDir string
}

// NewCustomFormatter creates a new formatter instance
func NewCustomFormatter(tf *tfexec.Terraform, workDir string) *CustomFormatter {
	return &CustomFormatter{tf: tf, workDir: workDir}
}

// findTerraformFiles finds all `.tf` files in the given directory
func (cf *CustomFormatter) findTerraformFiles(root string, recursive bool) ([]string, error) {
	var files []string

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".tf") && !strings.HasPrefix(filepath.Base(path), ".") {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			files = append(files, absPath)
		}

		if info.IsDir() && path != root && !recursive {
			return filepath.SkipDir
		}

		return nil
	}

	if err := filepath.Walk(root, walkFn); err != nil {
		return nil, err
	}
	return files, nil
}

// formatError formats a single formatting error in Terraform style
func (cf *CustomFormatter) formatError(err FormatError) string {
	var sb strings.Builder

	sb.WriteString("\n╷\n")
	sb.WriteString(fmt.Sprintf("│ %s%s\n", RedText("Error: "), err.Description))
	sb.WriteString("│\n")
	sb.WriteString(fmt.Sprintf("│   on %s line %d:\n", GreyText(err.Location), err.LineNumber))
	sb.WriteString(fmt.Sprintf("│   %d:   %s\n", err.LineNumber, err.LineContent))
	if err.NextLine != "" {
		sb.WriteString(fmt.Sprintf("│   %d:   %s\n", err.LineNumber+1, err.NextLine))
	}
	if err.HelpText != "" {
		sb.WriteString("│\n")
		sb.WriteString(fmt.Sprintf("│   %s\n", GreyText(err.HelpText)))
	}
	sb.WriteString("╵\n")
	return sb.String()
}

// FormatWithDetails performs Terraform formatting with rich logging
func (cf *CustomFormatter) FormatWithDetails(ctx context.Context, dir string, recursive bool) error {
	Info("Starting Terraform formatting process...")

	files, err := cf.findTerraformFiles(dir, recursive)
	if err != nil {
		Error("Failed to find Terraform files: %v", err)
		return fmt.Errorf("error finding Terraform files: %w", err)
	}

	if len(files) == 0 {
		Warn("No Terraform (.tf) files found in the directory.")
		return nil
	}

	formatted := []string{}
	var formatErrors []FormatError

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			formatErrors = append(formatErrors, FormatError{
				ErrorType:   "File read failed",
				Description: err.Error(),
				Location:    file,
				HelpText:    "Failed to read file for formatting check",
			})
			continue
		}

		fileDir := filepath.Dir(file)
		tf, err := tfexec.NewTerraform(fileDir, cf.tf.ExecPath())
		if err != nil {
			formatErrors = append(formatErrors, FormatError{
				ErrorType:   "Terraform init failed",
				Description: err.Error(),
				Location:    file,
				HelpText:    "Failed to initialize Terraform for formatting",
			})
			continue
		}

		var outputBuffer bytes.Buffer
		err = tf.Format(ctx, bytes.NewReader(content), &outputBuffer)
		if err != nil {
			formatErrors = append(formatErrors, cf.parseFormatError(err, file))
			continue
		}

		if bytes.Equal(content, outputBuffer.Bytes()) {
			continue // Already properly formatted
		}

		err = os.WriteFile(file, outputBuffer.Bytes(), 0644)
		if err != nil {
			formatErrors = append(formatErrors, FormatError{
				ErrorType:   "File write failed",
				Description: err.Error(),
				Location:    file,
				HelpText:    "Failed to write formatted content to file",
			})
			continue
		}

		formatted = append(formatted, file)
	}

	if len(formatErrors) > 0 {
		for _, e := range formatErrors {
			fmt.Print(cf.formatError(e))
		}
		Error("Formatting failed with %d errors", len(formatErrors))
		return fmt.Errorf("formatting failed with %d errors", len(formatErrors))
	}

	if len(formatted) > 0 {
		Success("Terraform files formatted successfully ✅")
		Info("Formatted files:")
		for _, file := range formatted {
			fmt.Printf("   %s\n", CyanText(file))
		}
	} else {
		Success("No Terraform file changes detected")
	}

	return nil
}

// parseFormatError converts terraform-exec errors into our FormatError type
func (cf *CustomFormatter) parseFormatError(err error, file string) FormatError {
	return FormatError{
		ErrorType:   "Format Error",
		Description: err.Error(),
		Location:    file,
		LineNumber:  1,
		HelpText:    "Please check file syntax and try again",
	}
}

// GetFmtTerraform initializes and returns a Terraform executor
func GetFmtTerraform() (*tfexec.Terraform, error) {
	workDir, err := os.Getwd()
	if err != nil {
		Error("Failed to get working directory: %v", err)
		return nil, fmt.Errorf("failed to get working directory: %w", err)
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

	return tf, nil
}

// Format applies canonical formatting to all Terraform files.
func Format(recursive bool) error {
	tf, err := GetFmtTerraform()
	if err != nil {
		return err
	}

	workDir, err := os.Getwd()
	if err != nil {
		Error("Failed to get working directory: %v", err)
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	formatter := NewCustomFormatter(tf, workDir)
	return formatter.FormatWithDetails(context.Background(), ".", recursive)
}
