package aws

import (
	awsCore "github.com/dfds/ce-cli/aws"
	"github.com/spf13/cobra"
)

var OrgAccountListCmd = &cobra.Command{
	Use:   "list-org-accounts",
	Short: "List Organization accounts",
	// Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.OrgAccountListCmd(cmd, args)
	},
}
