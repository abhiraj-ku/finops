package collector

import "time"

type ResourceType string

const (
	TypeEBSVolume   ResourceType = "EBS_VOLUME"
	TypeRDSInstance ResourceType = "RDS_INSTANCE"
)

type OrphanResource struct {
	ID                 string            `json:"id"`
	Type               ResourceType      `json:"type"`
	Region             string            `json:"region"`
	SizeGB             int32             `json:"size_gb,omitempty"`       // For EBS
	StorageType        string            `json:"storage_type,omitempty"`  // gp2, gp3, io2
	InstanceType       string            `json:"instance_type,omitempty"` // For RDS: db.t3.medium, etc.
	Status             string            `json:"status"`
	CreatedAt          time.Time         `json:"created_at"`
	Tags               map[string]string `json:"tags"`
	EstimatedMo        float64           `json:"estimated_monthly_cost_usd"`
	Creator            string            `json:"creator"`
	IsTerraformManaged bool              `json:"is_terraform_managed"`
	TerraformAddress   string            `json:"terraform_address"`
}
