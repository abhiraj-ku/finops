package collector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func UnattachedVolumes(ctx context.Context, client *ec2.Client, region string) ([]OrphanResource, error) {
	var results []OrphanResource

	// status available indicates unattached voulume
	i := &ec2.DescribeVolumesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("status"),
				Values: []string{"available"},
			},
		},
	}
	paginator := ec2.NewDescribeVolumesPaginator(client, i)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to descibe volume in %s: %w", region, err)
		}
		for _, vol := range page.Volumes {
			tags := make(map[string]string)
			for _, t := range vol.Tags {
				if t.Key != nil && t.Value != nil {
					tags[*t.Key] = *t.Value
				}
			}
			size := aws.ToInt32(vol.Size)
			volType := string(vol.VolumeType)

			// basline fallback cost calculation (gp3 ~$0.08/GB-month)
			ratePerGB := 0.08
			if volType == "io2" || volType == "io1" {
				ratePerGB = 0.125
			}
			estimatedCost := float64(size) * ratePerGB

			results = append(results, OrphanResource{
				ID:          aws.ToString(vol.VolumeId),
				Type:        TypeEBSVolume,
				Region:      region,
				SizeGB:      size,
				StorageType: volType,
				Status:      string(vol.State),
				CreatedAt:   aws.ToTime(vol.CreateTime),
				Tags:        tags,
				EstimatedMo: estimatedCost,
			})

		}
	}
	return results, nil
}
