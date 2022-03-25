package aws

import (
	awsCore "github.com/dfds/ce-cli/aws"
	"github.com/spf13/cobra"
)

var IamRoleCreateCmd = &cobra.Command{
	Use:   "create-iam-role",
	Short: "Create IAM role",
	// Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.CreateIAMRoleCmd(cmd, args)
	},
}

var IamRoleDeleteCmd = &cobra.Command{
	Use:   "delete-iam-role",
	Short: "Delete IAM role",
	// Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.DeleteIAMRoleCmd(cmd, args)
	},
}

func IamInit() {

	// define parameters input for IamRoleCreateCmd functionality
	IamRoleCreateCmd.PersistentFlags().StringP("role-name", "r", "", "Name to assign to the new role.")
	IamRoleCreateCmd.PersistentFlags().StringP("policy-name", "p", "", "Name to use for the Policy assigned to the new role.  If not provided the name of the Role will be reused.")
	IamRoleCreateCmd.PersistentFlags().StringP("policy-file", "f", "", "The path to a JSON file which holds the Policy document.")
	IamRoleCreateCmd.PersistentFlags().StringP("assumption-file", "a", "", "The path to a JSON file which holds the Role Assumption document.")
	IamRoleCreateCmd.PersistentFlags().StringP("role-description", "d", "", "A description that will be attached to the created Role.")
	IamRoleCreateCmd.PersistentFlags().StringP("policy-description", "o", "", "A description that will be attached to the created Policy.")
	IamRoleCreateCmd.PersistentFlags().Int32P("max-session-duration", "m", 3600, "The number of minutes that an assumed role will be valid for.")
	IamRoleCreateCmd.PersistentFlags().StringP("bucket-name", "b", "", "The name of an S3 Bucket where the Policy and Trust documents are held.")

	// define parameters input for IamRoleDeleteCmd functionality
	IamRoleDeleteCmd.PersistentFlags().StringP("role-name", "r", "", "Name to assign to the new role.")
	IamRoleDeleteCmd.PersistentFlags().StringP("policy-name", "p", "", "Name to use for the Policy assigned to the new role.  If not provided the name of the Role will be reused.")

}
