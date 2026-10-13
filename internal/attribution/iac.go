package attribution

import (
	"encoding/json"
	"fmt"
	"os"
)

// in this pasckage we are building quick hash map of
// AWS ID -> Terraform Address

type tfState struct {
	Resources []tfResources `json:"resources"`
}

type tfResources struct {
	Mode      string       `json:"mode"`
	Type      string       `json:"type"`
	Name      string       `json:"name"`
	Instances []tfInstance `json:"instance"`
}

type tfInstance struct {
	Attributes map[string]interface{} `json:"attributes"`
}

type IacInspector struct {
	resourceMap map[string]string // Maps AWS ID to Terraform Address (e.g. "vol-0a12" -> "aws_ebs_volume.data")
}

// loads the local .tf file and indexes the resources
func LoadState(filepath string) (*IacInspector, error) {
	fileByte, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("could not read the tfstate file: %w", err)

	}

	var state tfState

	if err := json.Unmarshal(fileByte, &state); err != nil {
		return nil, fmt.Errorf("failed to parse the tfstate json: %w", err)
	}

	idMap := make(map[string]string)

	for _, res := range state.Resources {
		// check only for the managed resources , not data sources
		if res.Mode != "managed" {
			continue
		}
		for _, instance := range res.Instances {
			// terraform stores the aws cloud id inside atrtributes["id"]
			if idVal, exists := instance.Attributes["id"]; exists {
				if awsId, ok := idVal.(string); ok && awsId != "" {
					// Construct the standard TF address (e.g., aws_db_instance.main)
					address := fmt.Sprintf("%s.%s", res.Type, res.Name)
					idMap[awsId] = address
				}
			}
		}
	}
	return &IacInspector{resourceMap: idMap}, nil
}

// check if AWS resoiuces exisits in the terraform state
func (iac *IacInspector) CheckResources(awsId string) (bool, string) {
	address, exists := iac.resourceMap[awsId]
	return exists, address
}
