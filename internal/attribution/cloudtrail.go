package attribution

import (
	"context"
	"time"

	"github.com/abhiraj-ku/finops/internal/collector"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
)

type Tracker struct {
	client *cloudtrail.Client
}

func NewTracker(client *cloudtrail.Client) *Tracker {
	return &Tracker{
		client: client,
	}
}

// find who created resources in last 90 dyays
func (t *Tracker) FindOwner(ctx context.Context, resource collector.OrphanResource) string {

	// last 90 days cloudtrail history

	startTime := time.Now().AddDate(0, 0, -90)

	input := &cloudtrail.LookupEventsInput{
		StartTime: aws.Time(startTime),
		LookupAttributes: []types.LookupAttribute{
			{
				AttributeKey:   types.LookupAttributeKeyResourceName,
				AttributeValue: aws.String(resource.ID),
			},
		},
	}
	paginator := cloudtrail.NewLookupEventsPaginator(t.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return "unknown (api error)"
		}

		for _, event := range page.Events {
			eventName := aws.ToString(event.EventName)

			isCreationEvent := false
			if resource.Type == collector.TypeEBSVolume && (eventName == "CreateVolume" || eventName == "RunInstances") {
				isCreationEvent = true
			}
			if resource.Type == collector.TypeRDSInstance && eventName == "CreateDBInstance" {
				isCreationEvent = true
			}

			if isCreationEvent {
				user := aws.ToString(event.Username)
				if user == "" {
					return "unknown (system role)"
				}
				return user
			}
		}
	}

	// If the loop finishes and finds nothing, it was created more than 90 days ago.
	return "unknown (>90 days old)"

}
