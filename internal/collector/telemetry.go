package collector

import "github.com/aws/aws-sdk-go-v2/service/cloudwatch"

type TelemetryAuditor struct {
	client *cloudwatch.Client
}

func NewTelemetryAuditor(client *cloudwatch.Client) *TelemetryAuditor {
	return &TelemetryAuditor{
		client: client,
	}
}

// sanitize creates a valid cloudwatch query id (starts with lowercase,alphanumeric only)
fund sanitizeID(prefix,id string) string{
	clean:= strings.map(func (r rune) rune  {
		if r>='a' && r<
		
	})
}
