package testhelpers

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

func SetupLocalStack(ctx context.Context) (aws.Config, func()) {
	lsContainer, err := localstack.Run(ctx,
		"localstack/localstack:3.0",
	)
	if err != nil {
		log.Fatalf("localstack setup failed: %v", err)
	}

	host, _ := lsContainer.Host(ctx)
	port, _ := lsContainer.MappedPort(ctx, "4566/tcp")
	endpoint := "http://" + host + ":" + port.Port()

	cfg, _ := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{AccessKeyID: "test", SecretAccessKey: "test"},
		}),
	)

	cfg.BaseEndpoint = aws.String(endpoint)

	return cfg, func() {
		lsContainer.Terminate(ctx)
	}
}
