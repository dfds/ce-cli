package aws

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"text/template"
)

var awsConfigTemplate = `
[profile {{.Name}}]
role_arn = arn:aws:iam::{{.Id}}:role/OrgRole
source_profile = saml

`

type awsConfigTemplateData struct {
	Id   string
	Name string
}

func AwsConfigGenerateCmd(cmd *cobra.Command, args []string) {
	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	excludeAccountIds, _ := cmd.Flags().GetStringSlice("exclude-account-ids")
	configTemplate := template.New("conf")
	configTemplateParsed, err := configTemplate.Parse(awsConfigTemplate)
	if err != nil {
		log.Fatal(err)
	}

	accountList, err := OrgAccountList(includeAccountIds, excludeAccountIds)
	if err != nil {
		fmt.Printf("Errr: %s\n", err)
	} else {
		for _, v := range accountList {
			var body bytes.Buffer
			err = configTemplateParsed.Execute(&body, awsConfigTemplateData{Name: *v.Name, Id: *v.Id})
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(body.String())
		}
	}
}
