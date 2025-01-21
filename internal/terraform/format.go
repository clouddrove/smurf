package terraform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// FormatError represents a single formatting error with its details
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
	return &CustomFormatter{
		tf:      tf,
		workDir: workDir,
	}
}

// findTerraformFiles finds all .tf files in the given directory
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

	err := filepath.Walk(root, walkFn)
	if err != nil {
		return nil, err
	}

	return files, nil
}


// formatError formats a single formatting error in Terraform style
func (cf *CustomFormatter) formatError(err FormatError) string {
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
	if err.NextLine != "" {
		sb.WriteString(fmt.Sprintf("%s   %d:   %s\n", errorSymbol, err.LineNumber+1, err.NextLine))
	}
	if err.HelpText != "" {
		sb.WriteString(fmt.Sprintf("%s\n", errorSymbol))
		sb.WriteString(fmt.Sprintf("%s %s\n", errorSymbol, err.HelpText))
	}
	sb.WriteString("╵\n")

	return sb.String()
}

// FormatWithDetails performs formatting and returns detailed output
func (cf *CustomFormatter) FormatWithDetails(ctx context.Context, dir string, recursive bool) error {
	spinner, _ := pterm.DefaultSpinner.Start("Formatting Terraform configuration files...")

	files, err := cf.findTerraformFiles(dir, recursive)
	if err != nil {
		spinner.Fail("Failed to find Terraform files")
		return fmt.Errorf("error finding Terraform files: %w", err)
	}

	formatted := make([]string, 0)
	var formatErrors []FormatError

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			formatErrors = append(formatErrors, FormatError{
				ErrorType:   "File read failed",
				Description: err.Error(),
				Location:    file,
				LineNumber:  0,
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
				LineNumber:  0,
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
			continue
		}

		err = os.WriteFile(file, outputBuffer.Bytes(), 0644)
		if err != nil {
			formatErrors = append(formatErrors, FormatError{
				ErrorType:   "File write failed",
				Description: err.Error(),
				Location:    file,
				LineNumber:  0,
				HelpText:    "Failed to write formatted content to file",
			})
			continue
		}

		formatted = append(formatted, file)
	}

	if len(formatErrors) > 0 {
		spinner.Fail(fmt.Sprintf("Formatting failed with %d errors", len(formatErrors)))
		for _, err := range formatErrors {
			fmt.Print(cf.formatError(err))
		}
		return fmt.Errorf("formatting failed with %d errors", len(formatErrors))
	}

	if len(formatted) > 0 {
		spinner.Success("Terraform files formatted successfully")
		pterm.Info.Println("\nFormatted files:")
		for _, file := range formatted {
			pterm.Info.Printf("- %s\n", file)
		}
	} else {
		spinner.Success("No files needed formatting")
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
		LineContent: "", 
		HelpText:    "Please check the file syntax and try again",
	}
}

func GetFmtTerraform() (*tfexec.Terraform, error) {
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

// Format applies a canonical format to Terraform configuration files.
// It runs `terraform fmt` in the current directory to ensure that all
// Terraform files adhere to the standard formatting conventions.
func Format(recursive bool) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	tf, err := GetFmtTerraform()
	if err != nil {
		return err
	}

	formatter := NewCustomFormatter(tf, workDir)
	return formatter.FormatWithDetails(context.Background(), ".", recursive)
}
