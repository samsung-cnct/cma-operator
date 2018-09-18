package cmagrpc

type GetClusterInput struct {
	Name     string
	Provider string
}

type GetClusterOutput struct {
	ID         string
	Name       string
	Status     string
	Kubeconfig string
}
