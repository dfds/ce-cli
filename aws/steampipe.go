package aws

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"strings"
	"text/template"
)

var steamPipeConfigTemplate = `
connection "role_aws_{{.Id}}" {
  plugin  = "aws"
  profile = "steampipe_{{.AwsProfile}}"
  regions = ["*"]
}
`

type steamPipeConfigTemplateData struct {
	AwsProfile string
	Id         string
}

func SteamPipeConfigGenerateCmd(cmd *cobra.Command, args []string) {
	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	excludeAccountIds, _ := cmd.Flags().GetStringSlice("exclude-account-ids")
	configTemplate := template.New("conf")
	configTemplateParsed, err := configTemplate.Parse(steamPipeConfigTemplate)
	if err != nil {
		log.Fatal(err)
	}

	aggregateAccountString := ""
	accountList, err := OrgAccountList(includeAccountIds, excludeAccountIds)
	if err != nil {
		fmt.Printf("Errr: %s\n", err)
	} else {
		for _, v := range accountList {
			var body bytes.Buffer
			err = configTemplateParsed.Execute(&body, steamPipeConfigTemplateData{AwsProfile: *v.Name, Id: *v.Id})
			if err != nil {
				log.Fatal(err)
			}

			aggregateAccountString = fmt.Sprintf("%s\"role_aws_%s\",", aggregateAccountString, *v.Id)

			fmt.Println(body.String())
		}
	}

	fmt.Printf(`
connection "all_aws_accounts" {
  plugin  = "aws"
  type    = "aggregator"
  connections = [%s]
}
`, strings.TrimSuffix(aggregateAccountString, ","))

}
