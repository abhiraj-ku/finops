package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type TelemetryAuditor struct {
	client *cloudwatch.Client
}

func NewTelemetryAuditor(client *cloudwatch.Client) *TelemetryAuditor {
	return &TelemetryAuditor{
		client: client,
	}
}

// sanitize creates a valid cloudwatch query id (starts with lowercase,alphanumeric only)
// prefix: category or identifie ("cpu")
// id : original identifier ("web-server-1")
func sanitizeID(prefix, id string) string {
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, id)
	return fmt.Sprintf("%s_%s", prefix, clean)
}

// filterRds checks database conns over last 14 days and returns data
func (a *TelemetryAuditor) FilterIdleRDS(ctx context.Context, candidates []OrphanResource) ([]OrphanResource, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -14)

	queries := make([]cwtypes.MetricDataQuery, 0, len(candidates))
	idToResource := make(map[string]OrphanResource)

	for _, db := range candidates {
		queryID := sanitizeID("rds", db.ID)
		idToResource[queryID] = db

		queries = append(queries, cwtypes.MetricDataQuery{
			Id: aws.String(queryID),
			MetricStat: &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{
					Namespace:  aws.String("AWS/RDS"),
					MetricName: aws.String("DatabaseConnections"),
					Dimensions: []cwtypes.Dimension{
						{
							Name:  aws.String("DBInstanceIdentifier"),
							Value: aws.String(db.ID),
						},
					},
				},
				Period: aws.Int32(86400), // 1 day aggregation
				Stat:   aws.String("Maximum"),
			},
			ReturnData: aws.Bool(true),
		})
	}

	input := &cloudwatch.GetMetricDataInput{
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
		MetricDataQueries: queries,
	}

	var idleResources []OrphanResource
	paginator := cloudwatch.NewGetMetricDataPaginator(a.client, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed fetching RDS metrics: %w", err)
		}

		for _, result := range output.MetricDataResults {
			queryID := aws.ToString(result.Id)
			resource, exists := idToResource[queryID]
			if !exists {
				continue
			}

			// Compute max connections observed across all returned data points
			var maxConns float64
			for _, val := range result.Values {
				if val > maxConns {
					maxConns = val
				}
			}

			// If max connections across 14 days is strictly 0 (or no metrics emitted because it was idle)
			if maxConns == 0 {
				idleResources = append(idleResources, resource)
			}
		}
	}

	return idleResources, nil
}

// FilterIdleEBS checks VolumeReadOps and VolumeWriteOps over 14 days to confirm 0 IOPS
func (a *TelemetryAuditor) FilterIdleEBS(ctx context.Context, candidates []OrphanResource) ([]OrphanResource, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -14)

	queries := make([]cwtypes.MetricDataQuery, 0, len(candidates)*2)
	queryMap := make(map[string]string) // queryID -> VolumeID

	for _, vol := range candidates {
		readID := sanitizeID("ebsr", vol.ID)
		writeID := sanitizeID("ebsw", vol.ID)
		queryMap[readID] = vol.ID
		queryMap[writeID] = vol.ID

		queries = append(queries,
			cwtypes.MetricDataQuery{
				Id: aws.String(readID),
				MetricStat: &cwtypes.MetricStat{
					Metric: &cwtypes.Metric{
						Namespace:  aws.String("AWS/EBS"),
						MetricName: aws.String("VolumeReadOps"),
						Dimensions: []cwtypes.Dimension{
							{Name: aws.String("VolumeId"), Value: aws.String(vol.ID)},
						},
					},
					Period: aws.Int32(86400),
					Stat:   aws.String("Sum"),
				},
				ReturnData: aws.Bool(true),
			},
			cwtypes.MetricDataQuery{
				Id: aws.String(writeID),
				MetricStat: &cwtypes.MetricStat{
					Metric: &cwtypes.Metric{
						Namespace:  aws.String("AWS/EBS"),
						MetricName: aws.String("VolumeWriteOps"),
						Dimensions: []cwtypes.Dimension{
							{Name: aws.String("VolumeId"), Value: aws.String(vol.ID)},
						},
					},
					Period: aws.Int32(86400),
					Stat:   aws.String("Sum"),
				},
				ReturnData: aws.Bool(true),
			},
		)
	}

	input := &cloudwatch.GetMetricDataInput{
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
		MetricDataQueries: queries,
	}

	activityPerVolume := make(map[string]float64)
	paginator := cloudwatch.NewGetMetricDataPaginator(a.client, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed fetching EBS metrics: %w", err)
		}

		for _, result := range output.MetricDataResults {
			volID := queryMap[aws.ToString(result.Id)]
			for _, val := range result.Values {
				activityPerVolume[volID] += val
			}
		}
	}

	var confirmedIdle []OrphanResource
	for _, vol := range candidates {
		if activityPerVolume[vol.ID] == 0 {
			confirmedIdle = append(confirmedIdle, vol)
		}
	}

	return confirmedIdle, nil
}
