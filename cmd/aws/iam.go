package aws

import (
	awsCore "github.com/dfds/ce-cli/aws"
	"github.com/spf13/cobra"
)

var PredefinedIamRoleCreateCmd = &cobra.Command{
	Use:   "create-predefined-iam-role",
	Short: "Create predefined IAM role",
	// Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.CreatePredefinedIAMRoleCmd(cmd, args)
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

var PredefinedIamRoleDeleteCmd = &cobra.Command{
	Use:   "delete-predefined-iam-role",
	Short: "Delete predefined IAM role",
	// Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.DeletePredefinedIAMRoleCmd(cmd, args)
	},
}

func IamInit() {

	PredefinedIamRoleCreateCmd.PersistentFlags().StringP("role-name", "r", "", "The name of a unique predefined role that will be deployed into the accounts specified.")
	PredefinedIamRoleCreateCmd.PersistentFlags().StringP("bucket-name", "b", "", "The name of an S3 Bucket where the Policy and Trust documents are held.")
	PredefinedIamRoleCreateCmd.PersistentFlags().StringP("bucket-role-arn", "", "", "The ARN of the role that will be used to access bucket contents.")

	PredefinedIamRoleDeleteCmd.PersistentFlags().StringP("role-name", "r", "", "Name to assign to the new role.")

	// define parameters input for IamRoleDeleteCmd functionality
	IamRoleDeleteCmd.PersistentFlags().StringP("role-name", "r", "", "Name to assign to the new role.")
	IamRoleDeleteCmd.PersistentFlags().StringP("policy-name", "p", "", "Name to use for the Policy assigned to the new role.  If not provided the name of the Role will be reused.")

}
