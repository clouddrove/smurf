package terraform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

	// Find files that need formatting
	filesNeedFormatting := []string{}
	timeoutReached := false
	processedCount := 0

	for _, file := range files {
		// Check if context has been cancelled (timeout reached)
		select {
		case <-ctx.Done():
			timeoutReached = true
			break
		default:
			// Continue processing
		}

		if timeoutReached {
			break
		}

		processedCount++

		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		fileDir := filepath.Dir(file)
		tf, err := tfexec.NewTerraform(fileDir, cf.tf.ExecPath())
		if err != nil {
			continue
		}

		var outputBuffer bytes.Buffer
		err = tf.Format(ctx, bytes.NewReader(content), &outputBuffer)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				timeoutReached = true
				break
			}
			continue
		}

		// Check if file needs formatting
		if !bytes.Equal(content, outputBuffer.Bytes()) {
			filesNeedFormatting = append(filesNeedFormatting, file)

			// Apply formatting if we haven't timed out yet
			if !timeoutReached {
				os.WriteFile(file, outputBuffer.Bytes(), 0644)
			}
		}
	}

	// Show files that need formatting
	if len(filesNeedFormatting) > 0 {
		fmt.Println("\nYou need to format following files:")
		for i, file := range filesNeedFormatting {
			relPath, err := filepath.Rel(cf.workDir, file)
			if err != nil {
				relPath = file
			}
			// Show numbering starting from 1
			fmt.Printf("  %d. %s\n", i+1, CyanText(relPath))
		}

		// Only show "formatted" message if we actually formatted them
		if !timeoutReached {
			Success("\nFormatted %d file(s).", len(filesNeedFormatting))
		}
	} else {
		Success("No Terraform files need formatting.")
	}

	// Show timeout message if reached
	if timeoutReached {
		fmt.Println() // Empty line before timeout message
		Warn("Timeout reached after processing %d/%d files. Some files may have been skipped.",
			processedCount, len(files))
	}

	return nil // Always return success
}

// parseFormatError converts terraform-exec errors into our FormatError type
func (cf *CustomFormatter) parseFormatError(err error, file string) FormatError {
	// Check if it's a timeout error from terraform-exec
	if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
		return FormatError{
			ErrorType:   "Processing Timeout",
			Description: "File processing took too long",
			Location:    file,
			LineNumber:  1,
			HelpText:    "This file may be very large or complex. Consider formatting it separately.",
		}
	}

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

// Format applies canonical formatting to all Terraform files with optional timeout.
func Format(recursive bool, timeout time.Duration) error {
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

	// Create context with timeout if specified
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()

		Info("Formatting with timeout: %v", timeout)
	}

	return formatter.FormatWithDetails(ctx, ".", recursive)
}
