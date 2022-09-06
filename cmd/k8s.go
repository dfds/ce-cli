package cmd

import (
	"github.com/dfds/ce-cli/cmd/k8s"
	"github.com/spf13/cobra"
)

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "K8s helper utilities",
	Run:   func(cmd *cobra.Command, args []string) {},
}

func k8sInit() {
	// Sandbox
	k8s.SandboxInit()
	k8sCmd.AddCommand(k8s.SandboxEksConfigGenerateCmd)
}
