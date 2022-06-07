package util

import (
	awsHttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"net/http"
)

// CreateHttpClientWithoutKeepAlive Currently the AWS SDK seems to let connections live for way too long. On OSes that has a very low file descriptior limit this becomes an issue.
func CreateHttpClientWithoutKeepAlive() *awsHttp.BuildableClient {
	client := awsHttp.NewBuildableClient().WithTransportOptions(func(transport *http.Transport) {
		transport.DisableKeepAlives = true
	})

	return client
}
