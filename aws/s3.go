package aws

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func DownloadS3File() {

	var awsS3Client *s3.Client

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-west-1"))
	if err != nil {
		log.Fatal(err)
	}

	awsS3Client = s3.NewFromConfig(cfg)

	// Name of the file where you want to save the downloaded file
	var filename string

	filename = "local-copy-policy.json"
	bucketName := "dfds-cloudengineering-cli"
	key := "aws/iam/inventory-role/policy.json"

	// bucketName = "dfds-pwes-k8s-public"
	// filename = "test.txt"
	// key = "kubeconfig/endor-saml.config"

	// Key to the file to be downloaded
	//var key string

	// Create the file
	newFile, err := os.Create(filename)
	if err != nil {
		log.Println(err)
	}
	defer newFile.Close()

	downloader := manager.NewDownloader(awsS3Client)
	_, err = downloader.Download(context.TODO(), newFile, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})

	fmt.Printf("err: %v\n", err)
}
