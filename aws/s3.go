package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/fatih/color"
)

type roleProperties struct {
	Description     string   `json:"description"`
	SessionDuration int32    `json:"sessionDuration"`
	Path            string   `json:"path"`
	ManagedPolicies []string `json:"managedpolicies"`
}

const DEFAULT_S3_BUCKET_KEY string = "aws/iam/"

var bucketName string

func DownloadIAMRoleFile(awsS3Client *s3.Client, roleName string, fileName string) []byte {

	// build the path to the properties file
	pfKey := fmt.Sprintf("%s%s-role/%s", DEFAULT_S3_BUCKET_KEY, roleName, fileName)

	// buffer and downloader to handle loading the file into memory
	buff := &manager.WriteAtBuffer{}
	downloader := manager.NewDownloader(awsS3Client)

	// get the file from the S3 bucket
	_, err := downloader.Download(context.TODO(), buff, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(pfKey),
	})

	// in the case of an error
	if err != nil {
		var ensk *s3types.NoSuchKey
		var ensb *s3types.NoSuchBucket

		// display suitable error depending on the nature of the issue
		if errors.As(err, &ensk) {
			color.Set(color.FgYellow)
			fmt.Printf("Configuration for a role named %s could not be retrieved.  Please verify that the provided role name is correct.\n", roleName)
			color.Set(color.FgWhite)
			os.Exit(1)
		}

		// this doesn't seem to work for some odd reason
		if errors.As(err, &ensb) {
			color.Set(color.FgYellow)
			fmt.Printf("Configuration for the role could not be retrieved.  Please ensure that you have provided the correct Bucket Name.\n")
			color.Set(color.FgWhite)
			os.Exit(1)
		}

		// default behaviour for errors we don't specifically handle
		color.Set(color.FgYellow)
		fmt.Println("An error occurred whilst trying to download the properties file from the S3 bucket.")
		fmt.Printf("The error was: %v\n", err)
		color.Set(color.FgWhite)
		os.Exit(1)
	}

	// return the downloaded data
	return buff.Bytes()

}

func DownloadRoleDocuments(bucketName string, bucketRoleArn string, roleName string) (roleProperties, string, string) {

	var trustPolicy string
	var inlinePolicy string
	var roleProperties roleProperties
	var awsS3Client *s3.Client

	// assume role required to access the CE-CLI S3 bucket
	creds, err := AssumeRole(bucketRoleArn)

	// if role assumption fails then...
	if err != nil {
		color.Set(color.FgYellow)
		fmt.Println("There was a problem while trying to assume the role required to access the CE CLI S3 bucket.  Please ensure that the Role ARN provided with the --bucket-role-arn parameter is set correctly.")
		fmt.Printf("The error was: %v\n", err)
		color.Set(color.FgWhite)
		os.Exit(1)
	} else {
		// create new configuration using assumed role credentials
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion("eu-west-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}

		// new s3 client
		awsS3Client = s3.NewFromConfig(cfg)

		// get byteslice of s3 document data
		buff := DownloadIAMRoleFile(awsS3Client, roleName, "properties.json")

		// unmarshall into the JSON struct
		err = json.Unmarshal(buff, &roleProperties)

		if err != nil {
			fmt.Println("The was a problem when trying to unmarshall the JSON data.")
			log.Fatalf("The error was: %v\n", err)
		}

		// retrieve trust policy for the role
		buff = DownloadIAMRoleFile(awsS3Client, roleName, "trust.json")
		trustPolicy = string(buff[:])

		// retrieve inline policy for the role
		buff = DownloadIAMRoleFile(awsS3Client, roleName, "inlinePolicy.json")
		inlinePolicy = string(buff[:])
	}

	// return properties and policy strings
	return roleProperties, trustPolicy, inlinePolicy
}
