package docker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/pterm/pterm"
)

func encodeAuthToBase64(authConfig registry.AuthConfig) (string, error) {
	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(authJSON), nil
}

func handleDockerResponse(responseBody io.ReadCloser, spinner *pterm.SpinnerPrinter, opts PushOptions) error {
	decoder := json.NewDecoder(responseBody)
	var lastProgress int
	for {
		var msg jsonmessage.JSONMessage
		if err := decoder.Decode(&msg); err == io.EOF {
			break
		} else if err != nil {
			pterm.Error.Println("Error decoding JSON:", err)
			return err
		}

		if msg.Error != nil {
			pterm.Error.Println("Error from Docker:", msg.Error.Message)
			return fmt.Errorf("%s", msg.Error.Message)
		}

		if msg.Progress != nil && msg.Progress.Total > 0 {
			current := int(msg.Progress.Current * 100 / msg.Progress.Total)
			if current > lastProgress {
				progressMessage := fmt.Sprintf("Pushing image %s... %d%%", opts.ImageName, current)
				spinner.UpdateText(progressMessage)
				fmt.Printf("\r%s", pterm.Green(progressMessage))
				lastProgress = current
			}
		}

		if msg.Stream != "" {
			fmt.Print(pterm.Blue(msg.Stream))
		}
	}

	spinner.Success("Image push complete.")
	link := fmt.Sprint("https://hub.docker.com/repository/")
	pterm.Info.Println("Image Pushed on Docker Hub:", link)
	pterm.Success.Println("Successfully pushed image:", opts.ImageName)
	return nil
}

func isM1Mac() bool {
	return runtime.GOOS == "darwin" && runtime.GOARCH == "arm64"
}

func createOptimizedTarArchive(contextDir string) (io.Reader, error) {
	excludePatterns := []string{
		".git",
		"node_modules",
		".DS_Store",
		".idea",
	}

	tarOpts := &archive.TarOptions{
		ExcludePatterns: excludePatterns,
	}

	return archive.TarWithOptions(contextDir, tarOpts)
}

func validateBuildContext(contextDir string) error {
	info, err := os.Stat(contextDir)
	if err != nil {
		return fmt.Errorf("invalid context directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("context must be a directory")
	}
	return nil
}

func convertToInterfaceMap(args map[string]string) map[string]*string {
	result := make(map[string]*string, len(args))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for key, value := range args {
		wg.Add(1)
		go func(k, v string) {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()
			result[k] = &v
		}(key, value)
	}

	wg.Wait()
	return result
}
