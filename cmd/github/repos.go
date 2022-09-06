package github

import (
	githubCore "github.com/dfds/ce-cli/github"
	"github.com/spf13/cobra"
)

var FixRepoPermissionsCmd = &cobra.Command{
	Use:   "fix-repo-permissions",
	Short: "Makes sure a default GitHub team is added to repositories owned by /dfds",
	Run: func(cmd *cobra.Command, args []string) {
		githubCore.FixRepoPermissionsCmd(cmd, args)
	},
}

func ReposInit() {

}
