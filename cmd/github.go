package cmd

import (
	"github.com/dfds/ce-cli/cmd/github"
	"github.com/spf13/cobra"
)

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "GitHub tooling",
	Run:   func(cmd *cobra.Command, args []string) {},
}

func githubInit() {
	github.ReposInit()
	githubCmd.AddCommand(github.FixRepoPermissionsCmd)

}
