package aws

import "github.com/spf13/cobra"

func CreateIAMRoleCmd(cmd *cobra.Command, args []string) {
	CreateIAMRole()
}

func CreateIAMRole() {
	accounts := OrgAccountList()
	var ids []string
	for _, v := range accounts {
		ids = append(ids, *v.Id)
	}

	AssumeRoleMultipleAccounts(ids)
}
