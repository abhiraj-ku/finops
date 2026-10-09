package awsclient

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

type Client struct {
	EC2        *ec2.Client
	RDS        *rds.Client
	CloudWatch *cloudwatch.Client
	Region     string
}

func New(ctx context.Context, region string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	return &Client{
		EC2:        ec2.NewFromConfig(cfg),
		RDS:        rds.NewFromConfig(cfg),
		CloudWatch: cloudwatch.NewFromConfig(cfg),
		Region:     region,
	}, nil

}
