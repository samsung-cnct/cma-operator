package sdsapplication

import (
	"github.com/samsung-cnct/cma-operator/pkg/util/cma"
	"github.com/samsung-cnct/cma-operator/pkg/util/sds/callback"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	api "github.com/samsung-cnct/cma-operator/pkg/apis/cma/v1alpha1"
	"github.com/samsung-cnct/cma-operator/pkg/util/helmutil"
	"github.com/samsung-cnct/cma-operator/pkg/util/k8sutil"
	"k8s.io/client-go/kubernetes"
)

func (c *SDSApplicationController) deployApplication(application *api.SDSApplication) (bool, error) {
	packageManager, err := cma.GetSDSPackageManager(application.Spec.PackageManager.Name+"-"+application.Spec.Cluster.Name, viper.GetString(KubernetesNamespaceViperVariableName), nil)
	if err != nil {
		logger.Infof("Error trying to install -->%s<-- could not get package manger becuase of %s", application.Spec.Name, err)
		return false, err
	}
	if packageManager.Status.Phase != api.PackageManagerPhaseImplemented {
		logger.Infof("Could not deploy app -->%s<-- as package manager is not ready, in phase -->%s<--", application.Spec.Name, packageManager.Status.Phase)
		return false, err
	}
	config, err := c.getRestConfigForRemoteCluster(application.Spec.Cluster.Name, application.Namespace, nil)
	if err != nil {
		return false, err
	}

	k8sutil.CreateJob(helmutil.GenerateHelmInstallJob(application.Spec), packageManager.Spec.Namespace, config)

	if application.Annotations[ClusterCallbackURLAnnotation] != "" {
		// We need to notify someone that the package manager is being deployed(again)
		message := &sdscallback.ClusterMessage{
			State:        sdscallback.ClusterMessageStateInProgress,
			StateText:    api.ApplicationPhaseInstalling,
			ProgressRate: 0,
		}
		sdscallback.DoCallback(application.Annotations[ClusterCallbackURLAnnotation], message)
	}

	application.Status.Phase = api.ApplicationPhaseInstalling
	_, err = c.client.CmaV1alpha1().SDSApplications(application.Namespace).Update(application)
	if err == nil {
		logger.Infof("Deployed helm install job for -->%s<--", application.Spec.Name)
	} else {
		logger.Infof("Could not update the status error was %s", err)
	}

	return true, nil
}

func (c *SDSApplicationController) waitForApplication(application *api.SDSApplication) (result bool, err error) {
	config, err := c.getRestConfigForRemoteCluster(application.Spec.Cluster.Name, application.Namespace, nil)
	if err != nil {
		return false, err
	}

	packageManager, err := cma.GetSDSPackageManager(application.Spec.PackageManager.Name+"-"+application.Spec.Cluster.Name, viper.GetString(KubernetesNamespaceViperVariableName), nil)
	if err != nil {
		logger.Infof("Cannot retrieve package manager for application %s", application.Spec.Name)
		return false, err
	}

	clientset, _ := kubernetes.NewForConfig(config)
	timeout := 0
	for timeout < 2000 {
		job, err := clientset.BatchV1().Jobs(packageManager.Spec.Namespace).Get("app-install-"+application.Spec.Name, v1.GetOptions{})
		if err == nil {
			if job.Status.Succeeded > 0 {
				application.Status.Phase = api.ApplicationPhaseImplemented
				application.Status.Ready = true
				_, err = c.client.CmaV1alpha1().SDSApplications(application.Namespace).Update(application)
				if err == nil {
					logger.Infof("Helm installed app -->%s<--", application.Spec.Name)
					c.handleInstalledApp(application)
				} else {
					logger.Infof("Could not update the status error was %s", err)
				}
				return true, nil
			}
		}
		time.Sleep(5 * time.Second)
		timeout++
	}
	return false, nil
}

func (c *SDSApplicationController) handleInstalledApp(application *api.SDSApplication) (result bool, err error) {
	if application.Annotations[ClusterCallbackURLAnnotation] != "" {
		// We need to notify someone that the package manager is being deployed(again)
		message := &sdscallback.ClusterMessage{
			State:        sdscallback.ClusterMessageStateCompleted,
			StateText:    api.ApplicationPhaseImplemented,
			ProgressRate: 100,
		}
		sdscallback.DoCallback(application.Annotations[ClusterCallbackURLAnnotation], message)
	}
	return true, nil
}
