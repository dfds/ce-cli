package aws

import (
	"fmt"

	"github.com/spf13/cobra"
)

func OrgAccountListCmd(cmd *cobra.Command, args []string) {
	OrgAccountList()
}

func OrgAccountList() {
	fmt.Println("aws list-org-accounts")
}
