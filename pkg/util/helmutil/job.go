package helmutil

import (
	"encoding/base64"
	sdsapi "github.com/samsung-cnct/cma-operator/pkg/apis/cma/v1alpha1"
	"github.com/samsung-cnct/cma-operator/pkg/util/cma"
	"github.com/samsung-cnct/cma-operator/pkg/util/k8sutil"
	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	KubernetesNamespaceViperVariableName = "kubernetes-namespace"
)

type TillerInitOptions struct {
	Name           string
	Namespace      string
	ServiceAccount string
	Version        string
	BackoffLimit   int32
}

func GenerateTillerInitJob(options TillerInitOptions) batchv1.Job {
	jobSpec := batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "helm",
						Image:           "quay.io/venezia/helm:" + options.Version,
						ImagePullPolicy: "Always",
						Command: []string{
							"/helm",
							"init",
							"--force-upgrade",
							"--tiller-namespace",
							options.Namespace,
							"--service-account",
							options.ServiceAccount,
							"--skip-refresh",
						},
					},
				},
				RestartPolicy:      corev1.RestartPolicyOnFailure,
				ServiceAccountName: options.ServiceAccount,
			},
		},
		BackoffLimit: &options.BackoffLimit,
	}
	return k8sutil.GenerateJob(options.Name, jobSpec)
}

func GenerateHelmInstallJob(application sdsapi.SDSApplicationSpec) batchv1.Job {
	packageManager, _ := cma.GetSDSPackageManager(application.PackageManager.Name+"-"+application.Cluster.Name, viper.GetString(KubernetesNamespaceViperVariableName), nil)
	jobName := "app-install-" + application.Name
	backoffLimit := int32(500)
	commandString := ""
	commandString += "/helm init -c && "
	commandString += "/helm repo add " + application.Chart.Repository.Name + " " + application.Chart.Repository.URL + " && "
	commandString += "( /helm delete --purge " + application.Name + " --tiller-namespace " + packageManager.Spec.Namespace + " || true ) && "
	commandString += "echo $HELMVALUES | base64 -d > /helm.values && "
	commandString += "/helm repo update && "
	commandString += "/helm install --tiller-namespace " + packageManager.Spec.Namespace + " --name " + application.Name
	commandString += " --namespace " + application.Namespace + " --values /helm.values "
	commandString += application.Chart.Repository.Name + "/" + application.Chart.Name
	commandString += ""

	jobSpec := batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            jobName,
						Image:           "quay.io/venezia/helm:" + packageManager.Spec.Version,
						ImagePullPolicy: "Always",
						Command:         []string{"/bin/bash", "-c", commandString},
						Env: []corev1.EnvVar{
							{
								Name:  "HELMVALUES",
								Value: base64.StdEncoding.EncodeToString([]byte(application.Values)),
							},
						},
					},
				},
				RestartPolicy:      corev1.RestartPolicyOnFailure,
				ServiceAccountName: "tiller-sa",
			},
		},
		BackoffLimit: &backoffLimit,
	}
	job := k8sutil.GenerateJob(jobName, jobSpec)

	return job
}
