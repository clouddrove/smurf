package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var (
	version    = "v1.0.0"
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show the version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s\n", getLatestVersion())
		},
	}
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func getLatestVersion() string {
	url := "https://api.github.com/repos/clouddrove/smurf/releases/latest"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return version
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return version
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: Unable to fetch latest version. Status code:", resp.StatusCode)
		return version
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return version
	}

	var release githubRelease
	err = json.Unmarshal(body, &release)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return version
	}

	if release.TagName != "" {
		return release.TagName
	}

	return version
}

func init() {
	// Version flag  --version
	RootCmd.Flags().BoolP("version", "", false, "Show version information")

	// Chain the PersistentPreRun to handle version flag
	originalPersistentPreRun := RootCmd.PersistentPreRun
	RootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			fmt.Println(getLatestVersion())
			os.Exit(0)
		}
		if originalPersistentPreRun != nil {
			originalPersistentPreRun(cmd, args)
		}
	}

	// Add version command
	RootCmd.AddCommand(versionCmd)
}
