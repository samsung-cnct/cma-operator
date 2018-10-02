package sdscallback

import (
	"encoding/json"
)

const (
	ClusterMessageStateInProgress = "INPROGRESS"
	ClusterMessageStateCompleted  = "COMPLETED"
	ClusterMessageStateFailed     = "FAILED"
)

type ClusterMessage struct {
	ErrorCode    string `json:"errorCode"`
	ErrorDetail  string `json:"errorDetail"`
	ErrorText    string `json:"errorText"`
	ProgressRate int    `json:"progressRate"`
	State        string `json:"state"`
	StateText    string `json:"stateText"`
	Data         string `json:"data"`
}

type ClusterDataPayload struct {
	Kubeconfig       string `json:"kube_config"`
	ClusterStatus    string `json:"cluster_status"`
	CreationDateTime string `json:"creation_datetime"`
}

func (m *ClusterMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}
