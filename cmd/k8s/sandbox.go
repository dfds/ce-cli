package k8s

import (
	"fmt"
	"github.com/dfds/ce-cli/k8s"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var SandboxEksConfigGenerateCmd = &cobra.Command{
	Use:   "new-sandbox-config",
	Short: "For setting up a new sandbox Kubernetes cluster",
	Long:  `Creates a directory with the necessary Terraform & Terragrunt files to get started with a sandbox Kubernetes cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("Incorrect amount of arguments given. This commands expects 1 argument: 'cluster-name'. Example: 'ce k8s new-sandbox-config'")
			os.Exit(1)
		}
		clusterName := args[0]

		fmt.Printf("Generating sandbox Kubernetes cluster for %s\n", clusterName)
		err := k8s.GenerateConfig(k8s.GenerateConfigRequest{ClusterName: clusterName})
		if err != nil {
			log.Fatal(err)
		}
	},
}

func SandboxInit() {
	//	SandboxEksConfigGenerateCmd.PersistentFlags().String()

	//OrgAccountListCmd.PersistentFlags().StringP("bucket-name", "b", "", "The name of the backend S3 bucket for the CLI.")
	//OrgAccountListCmd.PersistentFlags().StringP("bucket-role-arn", "", "", "The ARN of the role that will be used to access bucket contents.")
	//cobra.MarkFlagRequired(OrgAccountListCmd.PersistentFlags(), "bucket-name")
	//cobra.MarkFlagRequired(OrgAccountListCmd.PersistentFlags(), "bucket-role-arn")

}
