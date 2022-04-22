package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/fatih/color"
)

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
			fmt.Printf("Key %s for role %s not found.\n", pfKey, roleName)
			color.Set(color.FgWhite)
			os.Exit(1)
		}

		// this doesn't seem to work for some odd reason
		if errors.As(err, &ensb) {
			color.Set(color.FgYellow)
			fmt.Printf("Bucket %s not found for role %s.\n", bucketName, roleName)
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
		buff = DownloadIAMRoleFile(awsS3Client, roleName, "policy.json")
		inlinePolicy = string(buff[:])
	}

	// return properties and policy strings
	return roleProperties, trustPolicy, inlinePolicy
}

func GetExcludeAccountIdsFromS3(bucketName string, bucketRoleArn string, bucketKey string, scope string) []string {

	var awsS3Client *s3.Client
	var excludeAccountsStruct excludeAccountsStruct

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

		buff := &manager.WriteAtBuffer{}
		downloader := manager.NewDownloader(awsS3Client)

		// get the file from the S3 bucket
		_, err = downloader.Download(context.TODO(), buff, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(bucketKey),
		})
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		}

		// unmarshall into the JSON struct
		err = json.Unmarshal(buff.Bytes(), &excludeAccountsStruct)
		if err != nil {
			fmt.Println("Error unmarshalling The was a problem when trying to unmarshall the JSON data.")
			log.Fatalf("The error was: %v\n", err)
		}

	}

	// Merge common excluded accounts with scope-specific exclusions
	excludeAccountsReflect := reflect.ValueOf(excludeAccountsStruct.Scopes)
	excludeScopes := excludeAccountsReflect.Type()
	var excludeAccountsAppend []string
	var excludeAccounts []string

	for i := 0; i < excludeAccountsReflect.NumField(); i++ {
		if excludeScopes.Field(i).Name == "Common" || excludeScopes.Field(i).Name == scope {
			excludeAccountsAppend = excludeAccountsReflect.Field(i).Interface().([]string)
			excludeAccounts = append(excludeAccounts, excludeAccountsAppend...)
		}
	}

	if len(excludeAccounts) > 0 {
		sort.Strings(excludeAccounts)
		fmt.Printf("Excluding accounts %s for the \"%s\" scope, based on file %s\n", strings.Join(excludeAccounts, ", "), scope, bucketKey)
	}

	// return excludeAccountIds
	return excludeAccounts

}
