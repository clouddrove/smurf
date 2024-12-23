package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
)

// Scout scans a Docker image for known vulnerabilities using 'docker scout cves'.
func Scout(dockerTag, sarifFile string) error {
	ctx := context.Background()

	args := []string{"scout", "cves", dockerTag}

	if sarifFile != "" {
		args = append(args, "--output", sarifFile)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	spinner, _ := pterm.DefaultSpinner.Start("Running 'docker scout cves'")
	defer spinner.Stop()

	err := cmd.Run()

	spinner.Stop()

	outStr := stdoutBuf.String()
	errStr := stderrBuf.String()

	if err != nil {
		pterm.Error.Println("Error running 'docker scout cves':", err)
		if errStr != "" {
			pterm.Error.Println(errStr)
		}
		return fmt.Errorf("failed to run 'docker scout cves': %w", err)
	}

	if outStr != "" {
		pterm.Info.Println("Docker Scout CVEs output:")
		fmt.Println(color.YellowString(outStr))
	}

	if sarifFile != "" {
		if _, err := os.Stat(sarifFile); err == nil {
			pterm.Success.Println("SARIF report saved to:", sarifFile)
		} else {
			pterm.Warning.Println("Expected SARIF report not found at:", sarifFile)
		}
	}

	pterm.Success.Println("Scan completed successfully.")
	return nil
}
