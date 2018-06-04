package main

import (
	"github.com/samsung-cnct/cma-operator/cmd/cma-operator/cmd"
)

func main() {
	//viper.SetEnvPrefix("clustermanagerapi")
	//replacer := strings.NewReplacer("-", "_")
	//viper.SetEnvKeyReplacer(replacer)
	//
	//// using standard library "flag" package
	//flag.Int("port", 9050, "Port to listen on")
	//flag.String("kubeconfig", "", "Location of kubeconfig file")
	//
	//pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	//pflag.Parse()
	//viper.BindPFlags(pflag.CommandLine)
	//
	//viper.AutomaticEnv()

	cmd.Execute()
}
