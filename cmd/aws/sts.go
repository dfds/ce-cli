package aws

import (
	awsCore "github.com/dfds/ce-cli/aws"
	"github.com/spf13/cobra"
)

var StsBulkSendEmailCmd = &cobra.Command{
	Use:   "email-bulk-send -d data.json -f template/s3-policy-deprecation.tpl",
	Short: "Send templated emails in bulk using SES",
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.StsBulkSendEmailCmd(cmd, args)
	},
}

func StsInit() {
	StsBulkSendEmailCmd.PersistentFlags().StringP("data", "d", "", "Path to json file containing template variables and various metadata")
	StsBulkSendEmailCmd.PersistentFlags().StringP("template", "f", "", "Path to message template file")
	StsBulkSendEmailCmd.PersistentFlags().BoolP("dry-run", "r", false, "Test templating, but don't actually send email")
	cobra.MarkFlagRequired(StsBulkSendEmailCmd.PersistentFlags(), "data")
	cobra.MarkFlagRequired(StsBulkSendEmailCmd.PersistentFlags(), "template")
}
