package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/dfds/ce-cli/util"
	"github.com/spf13/cobra"
	"log"
	"os"
	"text/template"
	"time"
)

type templateVars struct {
	Vars map[string]interface{}
}

type data struct {
	Title   string      `json:"title"`
	Entries []dataEntry `json:"entries"`
}

type dataEntry struct {
	Name   string                 `json:"name"`
	Emails []string               `json:"emails"`
	Values map[string]interface{} `json:"values"`
}

type sesRequest struct {
	Msg    string
	Title  string
	From   string
	Emails []string
}

func StsBulkSendEmailCmd(cmd *cobra.Command, args []string) {
	// get parameters from cobra
	dataPath, _ := cmd.Flags().GetString("data")
	templatePath, _ := cmd.Flags().GetString("template")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	data, err := deserialiseFromFile[data](dataPath)
	if err != nil {
		log.Fatal(err)
	}

	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		log.Fatal(err)
	}

	templateContainer := template.New("gen")
	templateParsed, err := templateContainer.Parse(string(templateData))
	if err != nil {
		log.Fatal("Unable to parse template file")
	}

	titleTemplateContainer := template.New("title")
	titleTemplateParsed, err := titleTemplateContainer.Parse(data.Title)
	if err != nil {
		log.Fatal("Unable to parse template file")
	}

	for _, entry := range data.Entries {
		var body bytes.Buffer
		entry.Values["Name"] = entry.Name
		err = templateParsed.Execute(&body, templateVars{Vars: entry.Values})
		if err != nil {
			log.Println("Unable to generate template")
			log.Fatal(err)
		}

		var titleBody bytes.Buffer
		err = titleTemplateParsed.Execute(&titleBody, templateVars{Vars: entry.Values})

		if dryRun {
			log.Println("DRY RUN")
			log.Println(entry)

			fmt.Print("Template rendered:\nSTART --\n\n")
			fmt.Printf("Title: %s\n", titleBody.String())
			fmt.Printf("Body: %s\n", body.String())
			fmt.Print("END --\n\n")
			continue
		}

		for _, email := range entry.Emails {
			err = sendEmail(context.Background(), sesRequest{
				Msg:    body.String(),
				Title:  titleBody.String(),
				From:   "noreply@dfds.cloud",
				Emails: []string{email},
			})

			if err != nil {
				log.Println(fmt.Sprintf("Failed sending email to %s for Capability %s", email, entry.Name))
				log.Println(err)
			} else {
				log.Println(fmt.Sprintf("Sent email to %s for Capability %s", email, entry.Name))
			}

			time.Sleep(time.Millisecond * 750) //TODO: Implement actual rate limiting system, for now this'll do
		}
	}

}

func sendEmail(ctx context.Context, req sesRequest) error {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-west-1"), config.WithHTTPClient(util.CreateHttpClientWithoutKeepAlive()))
	if err != nil {
		return err
	}

	sesClient := sesv2.NewFromConfig(cfg)

	input := &sesv2.SendEmailInput{
		FromEmailAddress: &req.From,
		Destination:      &types.Destination{BccAddresses: req.Emails},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Body: &types.Body{Text: &types.Content{
					Data: &req.Msg,
				}},
				Subject: &types.Content{
					Data: &req.Title,
				},
			},
		},
	}

	output, err := sesClient.SendEmail(ctx, input)
	if err != nil {
		fmt.Println(output)
		return err
	}

	return nil
}

func deserialiseFromFile[V any](path string) (*V, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var payload *V

	err = json.Unmarshal(bytes, &payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}
