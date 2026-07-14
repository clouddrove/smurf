package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ValidOutputFormat reports whether format is one of allowed. Used by
// read-only commands that support -o/--output to reject an unknown value
// with a clear error instead of silently falling back to the default.
func ValidOutputFormat(format string, allowed ...string) bool {
	for _, a := range allowed {
		if format == a {
			return true
		}
	}
	return false
}

// PrintJSON marshals v as indented JSON and writes it, and nothing else, to
// stdout. It is meant for the -o json path of read-only commands: callers
// must not emit any other stdout output (spinners, pterm messages, AI
// explanations) alongside it, so a pipeline consuming stdout only ever sees
// the JSON document.
func PrintJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// CreateYamlFile creates a YAML file with given name and content safely.
func CreateYamlFile(fileName, content string) error {
	filePath := filepath.Join(".", fileName)

	// Prevent overwriting an existing file
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("⚠️  %s already exists. Delete or rename it before creating a new one", fileName)
	}

	// Write YAML content. smurf.yaml can hold credentials, so keep it readable
	// only by the owner.
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return fmt.Errorf("❌ failed to create %s: %v", fileName, err)
	}

	fmt.Printf("✅ %s created successfully at %s\n", fileName, filePath)
	return nil
}
