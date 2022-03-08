package aws

import (
	"fmt"

	"github.com/spf13/cobra"
)

var IamRoleCreateCmd = &cobra.Command{
	Use:   "create-iam-role",
	Short: "Create IAM role",
	// Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("aws create-iam-role")
	},
}
