package helm

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
)

// InstallationError represents a structured installation error
type InstallationError struct {
	Stage     string
	Operation string
	Err       error
	Context   map[string]string
}

func (e *InstallationError) Error() string {
	return fmt.Sprintf("%s: %s failed: %v", e.Stage, e.Operation, e.Err)
}

func (e *InstallationError) Unwrap() error {
	return e.Err
}

// NewInstallationError creates a new structured installation error
func NewInstallationError(stage, operation string, err error, context ...map[string]string) *InstallationError {
	errorObj := &InstallationError{
		Stage:     stage,
		Operation: operation,
		Err:       err,
		Context:   make(map[string]string),
	}

	if len(context) > 0 {
		errorObj.Context = context[0]
	}

	return errorObj
}

// FormatError formats an error with clean tree structure
func FormatError(err error) string {
	if ie, ok := err.(*InstallationError); ok {
		var sb strings.Builder
		sb.WriteString(pterm.Red("📛 INSTALLATION FAILED\n"))
		sb.WriteString("├── Stage:     " + ie.Stage + "\n")
		sb.WriteString("├── Operation: " + ie.Operation + "\n")
		sb.WriteString("├── " + pterm.LightRed("Error:     "+ie.Err.Error()+"\n"))

		if len(ie.Context) > 0 {
			sb.WriteString("└── Context:\n")
			i := 0
			for k, v := range ie.Context {
				if i == len(ie.Context)-1 {
					sb.WriteString("    └── " + k + ": " + v + "\n")
				} else {
					sb.WriteString("    ├── " + k + ": " + v + "\n")
				}
				i++
			}
		}

		return sb.String()
	}

	// For non-InstallationError types, return simple format
	return "❌ Error: " + err.Error()
}
