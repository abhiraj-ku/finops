package collector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/rds"
)

func runningRds(ctx context.Context, client *rds.Client, region string) ([]OrphanResource, error) {
	var results []OrphanResource

	paginator := rds.NewDescribeDBInstancesPaginator(client, &rds.DescribeDBInstancesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe RDS instance in %s: %w", region, err)

		}

		for _, db := range page.DBInstances {
			status := aws.ToString(db.DBInstanceStatus)
			// imspect only running/available databases for idle telemetry
			if status != "available" {
				continue
			}

			tags := make(map[string]string)
			for _, t := range db.TagList {
				if t.Key != nil && t.Value != nil {
					tags[*t.Key] = *t.Value
				}
			}
			instanceClass := aws.ToString(db.DBInstanceClass)
			allocationStorage := aws.ToInt32(db.AllocatedStorage)

			// ballmark estimate of montly cost for standard dev

		}
	}
}
