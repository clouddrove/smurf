package stf

import (
	"sync"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

// provisionCmd orchestrates multiple Terraform operations (init, plan, apply, output)
// in a sequential flow, grouping them into one streamlined command. After successful
// initialization, planning, and applying of changes, it retrieves the final outputs 
// asynchronously and handles any errors accordingly.
var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Its the combination of init, plan, apply, output for Terraform",
	RunE: func(cmd *cobra.Command, args []string) error {
		var wg sync.WaitGroup
		errChan := make(chan error, 5) 

		if err := terraform.Init(); err != nil {
			return err
		}

		if err := terraform.Plan("", ""); err != nil {
			return err
		}

		if err := terraform.Apply(); err != nil {
			return err
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := terraform.Output(); err != nil {
				errChan <- err
			}
		}()

		wg.Wait()

		close(errChan)

		for err := range errChan {
			if err != nil {
				return err
			}
		}

		return nil
	},
	Example: `
	smurf stf provision
	`,
}

func init() {
	stfCmd.AddCommand(provisionCmd)
}
