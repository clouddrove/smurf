package selm

import (
	"fmt"
	"strings"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	searchRepo    bool
	searchVersion string
	searchMaxCols int
	devel         bool
)

// searchCmd facilitates searching Helm charts
var searchCmd = &cobra.Command{
	Use:   "search [KEYWORD]",
	Short: "Search for Helm charts.",
	Long: `Search for Helm charts in repositories.

Examples:
  # Search all charts containing "nginx"
  smurf selm search nginx

  # Search repositories for "redis" charts
  smurf selm search redis --repo

  # Search for specific version
  smurf selm search postgres --version 12.0.0

  # Search in specific repository
  smurf selm search mysql --repo-url https://charts.bitnami.com/bitnami

  # Search development versions
  smurf selm search myapp --devel

  # Debug mode
  smurf selm search wordpress --debug

  # Limit column width for better display
  smurf selm search elasticsearch --max-cols 80`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if configs.Debug {
			pterm.EnableDebugMessages()
			pterm.Println("=== DEBUG MODE ENABLED ===")
		}

		var keyword string
		if len(args) >= 1 {
			keyword = args[0]
			if configs.Debug {
				pterm.Printf("Search keyword from argument: %s\n", keyword)
			}
		}

		if configs.Debug {
			pterm.Printf("Configuration\n")
			pterm.Printf("  - Keyword: %s\n", keyword)
			pterm.Printf("  - Repository search: %t\n", searchRepo)
			pterm.Printf("  - Version: %s\n", searchVersion)
			pterm.Printf("  - Development versions: %t\n", devel)
			pterm.Printf("  - Max Columns: %d\n", searchMaxCols)
			pterm.Printf("  - Repo URL: %s\n", RepoURL)
		}

		var results []helm.SearchResult
		var err error

		if searchRepo {
			if configs.Debug {
				pterm.Println("Searching repositories...")
			}
			results, err = helm.HelmSearchRepo(keyword, searchVersion, RepoURL, configs.Debug)
		} else {
			if configs.Debug {
				pterm.Println("Searching all charts...")
			}
			results, err = helm.HelmSearchAll(keyword, searchVersion, configs.Debug)
		}

		if err != nil {
			return err
		}

		// Filter development versions if not requested
		if !devel {
			results = filterStableVersions(results)
		}

		printSearchResults(results, searchMaxCols)

		return nil
	},
}

// filterStableVersions filters out development versions (alpha, beta, pre-release)
func filterStableVersions(results []helm.SearchResult) []helm.SearchResult {
	var stableResults []helm.SearchResult

	for _, result := range results {
		// Skip versions that contain pre-release identifiers
		if !strings.ContainsAny(result.Version, "-abcdefghijklmnopqrstuvwxyz") {
			stableResults = append(stableResults, result)
		}
	}

	return stableResults
}

func printSearchResults(results []helm.SearchResult, maxCols int) {
	if len(results) == 0 {
		pterm.Info.Println("No charts found matching your search criteria.")
		return
	}

	title := "SEARCH RESULTS"
	pterm.DefaultHeader.WithFullWidth().Println(title)
	pterm.Println()

	// Create table data
	tableData := pterm.TableData{
		{"NAME", "VERSION", "APP VERSION", "DESCRIPTION"},
	}

	for _, result := range results {
		// Truncate description if needed
		description := result.Description
		if maxCols > 0 && len(description) > maxCols-60 { // Reserve space for other columns
			description = description[:maxCols-63] + "..."
		}

		tableData = append(tableData, []string{
			fmt.Sprintf("%s/%s", result.Repository, result.Name),
			result.Version,
			result.AppVersion,
			description,
		})
	}

	// Print table
	err := pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	if err != nil {
		// Fallback to simple output if table rendering fails
		pterm.Info.Printf("Found %d charts:\n", len(results))
		for _, result := range results {
			pterm.Printf("• %s/%s (v%s, app v%s) - %s\n",
				result.Repository, result.Name, result.Version, result.AppVersion, result.Description)
		}
	}

	pterm.Success.Printf("Found %d matching charts\n", len(results))
}

func init() {
	searchCmd.Flags().BoolVar(&searchRepo, "repo", false, "Search repository charts instead of all charts")
	searchCmd.Flags().StringVar(&searchVersion, "version", "", "Search for specific chart version")
	searchCmd.Flags().BoolVar(&devel, "devel", false, "Include development versions (alpha, beta, pre-release)")
	searchCmd.Flags().IntVar(&searchMaxCols, "max-cols", 120, "Maximum column width for description")
	searchCmd.Flags().BoolVar(&configs.Debug, "debug", false, "Enable verbose output")
	searchCmd.Flags().StringVar(&RepoURL, "repo-url", "", "Specific repository URL to search in")
	selmCmd.AddCommand(searchCmd)
}
