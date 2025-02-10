package aws

import (
	"bufio"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/resourceexplorer2"
	"github.com/aws/aws-sdk-go-v2/service/resourceexplorer2/types"
	"github.com/dfds/ce-cli/util"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var GetResourcesCmd = &cobra.Command{
	Use:   "resource-explorer",
	Short: "Get all AWS Resources",
	Run: func(cmd *cobra.Command, args []string) {
		GetResources(cmd, args)
	},
}

type ResourceRow struct {
	Region               string
	Service              string
	Arn                  string
	HasMandatoryTags     bool
	HasMandatoryProdTags bool
	HasAtLeastOneDfdsTag bool
	MandatoryTags        map[string]string
	MandatoryProdTags    map[string]string
}

var mandatoryTags = map[string]string{
	"dfds.owner":      "",
	"dfds.env":        "",
	"dfds.costCentre": "",
}

var mandatoryProdTags = map[string]string{
	"dfds.data.backup":           "",
	"dfds.data.backup_retention": "",
	"dfds.service.availability":  "",
}

func GetResources(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-central-1"), config.WithHTTPClient(util.CreateHttpClientWithoutKeepAlive()))
	if err != nil {
		log.Fatal(err)
	}

	resourceClient := resourceexplorer2.NewFromConfig(cfg)
	taggedResp, err := listResources(resourceClient, ctx, "tag:all", nil)
	if err != nil {
		log.Fatal(err)
	}

	untaggedResp, err := listResources(resourceClient, ctx, "tag:none -tag.key:aws*", nil)
	if err != nil {
		log.Fatal(err)
	}

	var rows []ResourceRow

	for _, resource := range taggedResp {
		//fmt.Printf("%s %s - %s\n", *resource.Service, *resource.ResourceType, *resource.Arn)
		for _, property := range resource.Properties {
			if *property.Name == "tags" {
				tags, err := convertAwsTagsResponseToMap(property)
				if err != nil {
					log.Fatal(err)
				}
				row := ResourceRow{
					Region:        *resource.Region,
					Service:       *resource.Service,
					Arn:           *resource.Arn,
					MandatoryTags: make(map[string]string),
				}

				isProdResource := false

				for mandatoryTag, _ := range mandatoryTags {
					if tagValue, ok := tags[mandatoryTag]; ok {
						row.HasAtLeastOneDfdsTag = true
						row.MandatoryTags[mandatoryTag] = tagValue

						if mandatoryTag == "dfds.env" && tagValue == "prod" {
							isProdResource = true
						}
					}
				}

				if len(row.MandatoryTags) == len(mandatoryTags) {
					row.HasMandatoryTags = true
				}

				if isProdResource {
					row.MandatoryProdTags = make(map[string]string)
					for mandatoryTag, _ := range mandatoryProdTags {
						if tagValue, ok := tags[mandatoryTag]; !ok {
							row.HasMandatoryTags = false
						} else {
							row.MandatoryProdTags[mandatoryTag] = tagValue
						}
					}
				}

				if len(row.MandatoryProdTags) == len(mandatoryProdTags) {
					row.HasMandatoryProdTags = true
				}

				rows = append(rows, row)
			}

		}
	}

	resourcesWithMandatoryTags := 0
	resourcesWithAtLeastOneDfdsTag := 0
	for _, row := range rows {
		if len(row.MandatoryTags) > 0 {
			fmt.Printf("%s\n", row.Arn)
			fmt.Printf("  hasMandatoryTags: %t\n", row.HasMandatoryTags)
			fmt.Printf("  hasMandatoryProdTags: %t\n", row.HasMandatoryProdTags)
			fmt.Printf("  hasAtLeastOneDfdsTag: %t\n", row.HasAtLeastOneDfdsTag)
			fmt.Printf("  MandatoryTags: %v\n", row.MandatoryTags)
		}
		if row.HasMandatoryProdTags {
			resourcesWithMandatoryTags += 1
		}
		if row.HasAtLeastOneDfdsTag {
			resourcesWithAtLeastOneDfdsTag += 1
		}
	}

	fmt.Printf("Tagged resources: %d\n", len(taggedResp))
	fmt.Printf("Resources with mandatory tags: %d\n", resourcesWithMandatoryTags)
	fmt.Printf("Resources with at least one DFDS tag: %d\n", resourcesWithAtLeastOneDfdsTag)
	fmt.Printf("Untagged resources: %d\n", len(untaggedResp))

	err = saveToCsv(rows, "data.csv")
	if err != nil {
		log.Fatal(err)
	}
}

func saveToCsv(data []ResourceRow, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)

	writer.WriteString("region,service,arn,hasMandatoryTags,hasMandatoryProdTags,hasAtLeastOneDfdsTag,MandatoryTags,mandatoryProdTags\n")

	for _, row := range data {
		writer.WriteString(fmt.Sprintf("%s,%s,%s,%t,%t,%t,%v,%v\n", row.Region, row.Service, row.Arn, row.HasMandatoryTags, row.HasMandatoryProdTags, row.HasAtLeastOneDfdsTag, row.MandatoryTags, row.MandatoryProdTags))
	}

	writer.Flush()

	return nil
}

func convertAwsTagsResponseToMap(property types.ResourceProperty) (map[string]string, error) {
	var tags map[string]string = make(map[string]string)

	var propertyTags []map[string]string
	err := property.Data.UnmarshalSmithyDocument(&propertyTags)
	if err != nil {
		return tags, err
	}
	for _, tag := range propertyTags {
		tags[tag["Key"]] = tag["Value"]
	}

	return tags, nil
}

func listResources(client *resourceexplorer2.Client, ctx context.Context, query string, nextToken *string) ([]types.Resource, error) {
	resp, err := client.ListResources(ctx, &resourceexplorer2.ListResourcesInput{
		Filters: &types.SearchFilter{
			FilterString: aws.String(query),
		},
		MaxResults: aws.Int32(999),
		NextToken:  nextToken,
	})
	if err != nil {
		return nil, err
	}

	var resources []types.Resource
	resources = append(resources, resp.Resources...)

	if resp.NextToken != nil {
		rResp, err := listResources(client, ctx, query, resp.NextToken)
		if err != nil {
			return nil, err
		}
		resources = append(resources, rResp...)
	}

	return resources, nil
}
