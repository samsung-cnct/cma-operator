package cmagrpc

import (
	"context"
	"crypto/tls"
	pb "github.com/samsung-cnct/cluster-manager-api/pkg/generated/api"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.ClusterClient
}

func CreateNewClient(hostname string, insecure bool) (ClientInterface, error) {
	output := Client{}
	err := output.CreateNewClient(hostname, insecure)
	if err != nil {
		return nil, err
	}
	return &output, err
}

func CreateNewDefaultClient() (ClientInterface, error) {
	return CreateNewClient(viper.GetString("cma-endpoint"), viper.GetBool("cma-insecure"))
}

func (a *Client) CreateNewClient(hostname string, insecure bool) error {
	var err error
	if insecure {
		a.conn, err = grpc.Dial(hostname, grpc.WithInsecure())
		if err != nil {
			return err
		}
	} else {
		// If TLS is enabled, we're going to create credentials, also using built in certificates
		var tlsConf tls.Config
		creds := credentials.NewTLS(&tlsConf)

		a.conn, err = grpc.Dial(hostname, grpc.WithTransportCredentials(creds))
		if err != nil {
			return err
		}
	}
	a.client = pb.NewClusterClient(a.conn)
	return nil
}

func (a *Client) Close() error {
	return a.conn.Close()
}

func (a *Client) SetClient(client pb.ClusterClient) {
	a.client = client
}

func (a *Client) GetCluster(input GetClusterInput) (GetClusterOutput, error) {
	result, err := a.client.GetCluster(context.Background(), &pb.GetClusterMsg{
		Name:     input.Name,
		Provider: pb.Provider(pb.Provider_value[input.Provider]),
	})
	if err != nil {
		return GetClusterOutput{}, err
	}
	output := GetClusterOutput{
		ID:          result.Cluster.Id,
		Name:        result.Cluster.Name,
		Status:      result.Cluster.Status.String(),
		Kubeconfig:  result.Cluster.Kubeconfig,
		Bearertoken: result.Cluster.Bearertoken,
	}
	return output, nil
}
