package docker

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/pterm/pterm"
)

func warnIfContextLarge(contextDir string) {
	var total int64
	_ = filepath.Walk(contextDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	sizeMB := float64(total) / (1024 * 1024)
	pterm.Info.Printf("📦 Build context size: %.2f MB\n", sizeMB)
	if sizeMB > 100 {
		pterm.Warning.Println("⚠️ Build context is large. Consider using .dockerignore to exclude unnecessary files.")
	}
}

// Build builds a Docker image with identical output in local & CI
func Build(imageName, tag string, opts BuildOptions) error {
	// Make pterm CI-friendly if running in GitHub Actions
	if os.Getenv("CI") == "true" {
		pterm.DisableStyling()
	}

	start := time.Now()
	spinner, _ := pterm.DefaultSpinner.Start("Initializing build...")

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		spinner.Fail()
		return fmt.Errorf("docker client init failed: %v", err)
	}
	defer cli.Close()

	// Context size warning
	warnIfContextLarge(opts.ContextDir)

	// Prepare build context
	buildCtx, err := createTarball(opts.ContextDir, []string{".git", "node_modules"})
	if err != nil {
		spinner.Fail()
		return fmt.Errorf("context creation failed: %v", err)
	}
	defer buildCtx.Close()

	relDockerfilePath, err := filepath.Rel(opts.ContextDir, opts.DockerfilePath)
	if err != nil {
		spinner.Fail()
		return fmt.Errorf("invalid dockerfile path: %v", err)
	}

	buildArgsPtr := map[string]*string{}
	for k, v := range opts.BuildArgs {
		val := v
		buildArgsPtr[k] = &val
	}

	fullImageName := fmt.Sprintf("%s:%s", imageName, tag)

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{fullImageName},
		Dockerfile: relDockerfilePath,
		NoCache:    opts.NoCache,
		Remove:     true,
		BuildArgs:  buildArgsPtr,
		Target:     opts.Target,
		Platform:   opts.Platform,
		BuildID:    fmt.Sprintf("build-%d", time.Now().Unix()),
		PullParent: true,
	}

	// Enable BuildKit if requested
	if opts.BuildKit {
		os.Setenv("DOCKER_BUILDKIT", "1")
	}

	spinner.Success()

	// Start progress bar
	progressBar, _ := pterm.DefaultProgressbar.WithTotal(100).WithTitle("Building").Start()

	// Use Docker API for both BuildKit & classic builds
	resp, err := cli.ImageBuild(ctx, buildCtx, buildOptions)
	if err != nil {
		progressBar.Stop()
		return fmt.Errorf("build failed: %v", err)
	}
	defer resp.Body.Close()

	// Process Docker build messages
	decoder := json.NewDecoder(resp.Body)
	for {
		var msg jsonmessage.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			progressBar.Stop()
			return fmt.Errorf("decode build output failed: %v", err)
		}

		if msg.Error != nil {
			progressBar.Stop()
			return fmt.Errorf("build error: %v", msg.Error)
		}

		if msg.Stream != "" {
			if strings.HasPrefix(msg.Stream, "Step ") {
				pterm.Info.Print(msg.Stream)
				progressBar.Add(5)
			} else if trimmed := strings.TrimSpace(msg.Stream); trimmed != "" {
				pterm.Debug.Println(trimmed)
			}
		}
		if msg.Status != "" {
			progressBar.Add(1)
		}
	}

	progressBar.Add(100 - progressBar.Current)

	// Inspect built image
	inspect, err := cli.ImageInspect(ctx, fullImageName)
	if err != nil {
		return fmt.Errorf("failed to get image info: %v", err)
	}

	panel := pterm.DefaultBox.WithTitle("Build Complete").Sprintf(` %s %s %s %s %s %s %s `,
		pterm.FgCyan.Sprintf("✓ Image Built Successfully"),
		pterm.FgCyan.Sprintf("Image: %s", fullImageName),
		pterm.FgCyan.Sprintf("ID: %s", inspect.ID[:12]),
		pterm.FgCyan.Sprintf("Size: %.2f MB", float64(inspect.Size)/1024/1024),
		pterm.FgCyan.Sprintf("Platform: %s/%s", inspect.Os, inspect.Architecture),
		pterm.FgCyan.Sprintf("Created: %s", inspect.Created[:19]),
		pterm.FgCyan.Sprintf("Layers: %d", len(inspect.RootFS.Layers)),
	)

	fmt.Println(panel)
	pterm.Info.Printf("⏱️  Total Build Duration: %s\n", time.Since(start).Round(time.Second))
	return nil
}

// createTarball now logs skips as debug instead of errors
func createTarball(srcDir string, excludePatterns []string) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	tw := tar.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer tw.Close()

		filepath.Walk(srcDir, func(file string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, _ := filepath.Rel(srcDir, file)

			// Skip excluded patterns
			for _, pattern := range excludePatterns {
				if strings.HasPrefix(relPath, pattern) {
					if fi.IsDir() {
						pterm.Debug.Println("Skipping directory:", relPath)
						return filepath.SkipDir
					}
					pterm.Debug.Println("Skipping file:", relPath)
					return nil
				}
			}

			if relPath == "." {
				return nil
			}

			hdr, err := tar.FileInfoHeader(fi, relPath)
			if err != nil {
				return err
			}
			hdr.Name = relPath

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			if fi.Mode().IsRegular() {
				f, err := os.Open(file)
				if err != nil {
					return err
				}
				defer f.Close()
				if _, err := io.Copy(tw, f); err != nil {
					return err
				}
			}
			return nil
		})
	}()
	return pr, nil
}
