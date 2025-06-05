package docker

import (
	"encoding/base64"
	"encoding/json"

	"github.com/docker/docker/api/types/registry"
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
