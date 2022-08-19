package aws

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"text/template"
)

var steamPipeConfigTemplate = `
connection "role_aws_{{.AwsProfile}}" {
  plugin  = "aws"
  profile = "steampipe_{{.AwsProfile}}"
  regions = ["*"]
}
`

type steamPipeConfigTemplateData struct {
	AwsProfile string
}

func SteamPipeConfigGenerateCmd(cmd *cobra.Command, args []string) {
	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	excludeAccountIds, _ := cmd.Flags().GetStringSlice("exclude-account-ids")
	configTemplate := template.New("conf")
	configTemplateParsed, err := configTemplate.Parse(steamPipeConfigTemplate)
	if err != nil {
		log.Fatal(err)
	}

	accountList, err := OrgAccountList(includeAccountIds, excludeAccountIds)
	if err != nil {
		fmt.Printf("Errr: %s\n", err)
	} else {
		for _, v := range accountList {
			var body bytes.Buffer
			err = configTemplateParsed.Execute(&body, steamPipeConfigTemplateData{AwsProfile: *v.Name})
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(body.String())
		}
	}
}
