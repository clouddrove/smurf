package sdkr

import (
	"bufio"
	"errors"
	"os"
	"strings"

	"github.com/clouddrove/smurf/configs"
	"github.com/pterm/pterm"
	"golang.org/x/term"
)

// confirmPush asks the user to confirm before pushing an image. It is shared
// by all sdkr provision commands so their prompt behavior stays identical.
//
// It does nothing (returns nil immediately) when --yes was passed, or when
// stdin is not a TTY (e.g. a CI pipeline with stdin closed or redirected),
// so automated runs never hang waiting for input. When it does prompt and
// the user answers anything other than "y"/"Y", it returns an error so the
// caller aborts cleanly instead of pushing.
func confirmPush() error {
	if configs.ConfirmAfterPush {
		return nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil
	}

	pterm.Info.Print("Proceed with push? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)
	if response != "y" && response != "Y" {
		return errors.New("push aborted by user")
	}
	return nil
}
