package collector

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/rds"
)

func RunningRdsInfo(ctx context.Context, client *rds.Client, region string) ([]OrphanResource, error) {
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
			allocatedStorage := aws.ToInt32(db.AllocatedStorage)

			// ballmark estimate of montly cost for standard dev
			cost := estimaterRdsCost(instanceClass, allocatedStorage)

			results = append(results, OrphanResource{
				ID:           aws.ToString(db.DBInstanceIdentifier),
				Type:         TypeRDSInstance,
				Region:       region,
				InstanceType: instanceClass,
				SizeGB:       allocatedStorage,
				Status:       status,
				CreatedAt:    aws.ToTime(db.InstanceCreateTime),
				Tags:         tags,
				EstimatedMo:  cost,
			})

		}
	}
	return results, nil
}

// cost estimation function
func estimaterRdsCost(class string, storageGb int32) float64 {
	storageCost := float64(storageGb) * 0.115
	var computeCost float64

	switch {
	case strings.Contains(class, "micro"):
		computeCost = 13.00
	case strings.Contains(class, "small"):
		computeCost = 26.00
	case strings.Contains(class, "medium"):
		computeCost = 52.00
	case strings.Contains(class, "large"):
		computeCost = 104.00
	case strings.Contains(class, "xlarge"):
		computeCost = 208.00
	default:
		computeCost = 50.00
	}

	return computeCost + storageCost

}
