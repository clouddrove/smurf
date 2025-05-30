package docker

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types/registry"
	"github.com/pterm/pterm"
)

// encodeAuthToBase64 encodes the given registry.AuthConfig as a base64-encoded string.
// used in the push function to authenticate with the Docker registry.
func encodeAuthToBase64(authConfig registry.AuthConfig) (string, error) {
	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(authJSON), nil
}

// logAndReturnError logs the error to the console and returns a formatted error.
func logAndReturnError(format string, args ...any) error {
	errMsg := fmt.Sprintf(format, args...)
	pterm.Error.Println(errMsg)
	return errors.New(errMsg)
}
