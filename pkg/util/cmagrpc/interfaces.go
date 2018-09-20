package cmagrpc

import (
	pb "github.com/samsung-cnct/cluster-manager-api/pkg/generated/api"
)

type ClientInterface interface {
	GetCluster(GetClusterInput) (GetClusterOutput, error)
	CreateNewClient(string, bool) error
	Close() error
	SetClient(client pb.ClusterClient)
}
