package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type roleProperties struct {
	Description     string   `json:"description"`
	SessionDuration int64    `json:"sessionDuration"`
	Path            string   `json:"path"`
	ManagedPolicies []string `json:"managedpolicies"`
}

const DEFAULT_S3_BUCKET_KEY string = "aws/iam/"

func DownloadS3File(bucketName string, bucketRoleArn string, roleName string) {

	var roleProperties roleProperties
	var awsS3Client *s3.Client

	// assume role required to access the CE-CLI S3 bucket
	creds, err := AssumeRole(bucketRoleArn)

	// if role assumption fails then...
	if err != nil {
		fmt.Println("There was a problem while trying to assume the role required to access the CE CLI S3 bucket.")
		log.Fatalf("The error was: %v\n", err)
	} else {
		// create new configuration using assumed role credentials
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion("eu-west-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}

		// new s3 client
		awsS3Client = s3.NewFromConfig(cfg)

		// build the path to the properties file
		pfKey := fmt.Sprintf("%s%s/properties.json", DEFAULT_S3_BUCKET_KEY, roleName)

		// buffer and downloader to handle loading the file into memory
		buff := &manager.WriteAtBuffer{}
		downloader := manager.NewDownloader(awsS3Client)

		// get the file from the S3 bucket
		numBytes, err := downloader.Download(context.TODO(), buff, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(pfKey),
		})

		// unmarshall into the JSON struct
		err = json.Unmarshal(buff.Bytes(), &roleProperties)

		if err != nil {
			fmt.Println("The was a problem when trying to unmarshall the JSON data.")
			log.Fatalf("The error was: %v\n", err)
		} else {
			fmt.Println("numBytes: ", numBytes)
			fmt.Printf("Description: %s\n", roleProperties.Description)
			fmt.Printf("Session Duration: %v\n", roleProperties.SessionDuration)
			fmt.Printf("Path: %s\n", roleProperties.Path)
			for _, v := range roleProperties.ManagedPolicies {
				fmt.Printf("Managed Policy: %s\n", v)
			}
		}
	}
}
