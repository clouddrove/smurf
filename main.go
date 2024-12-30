/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/clouddrove/smurf/cmd"
	_ "github.com/clouddrove/smurf/cmd/stf"
	_ "github.com/clouddrove/smurf/cmd/selm"
	_ "github.com/clouddrove/smurf/cmd/sdkr"
)

// main is the entry point for the CLI.
func main() {
	cmd.Execute()
}
