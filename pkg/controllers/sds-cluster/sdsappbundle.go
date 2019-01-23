package sds_cluster

import (
	api "github.com/samsung-cnct/cma-operator/pkg/apis/cma/v1alpha1"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeSchema "k8s.io/apimachinery/pkg/runtime/schema"
)

// checkBundles
func (c *SDSClusterController) checkBundles(cluster *api.SDSCluster) error {

	// get all sdsappbundles
	bundleList, err := c.client.CmaV1alpha1().SDSAppBundles(viper.GetString(KubernetesNamespaceViperVariableName)).List(v1.ListOptions{})
	if err != nil {
		logger.Errorf("could not get list of SDSAppBundles, error: %s", err)
		return err
	}

	for _, bundle := range bundleList.Items {
		// check if provider specific
		if len(bundle.Spec.Providers) == 0 {
			// install on all providers
			err := c.createBundlePackageManager(&bundle, cluster)
			if err != nil {
				return err
			}

			err = c.createBundleApplications(&bundle, cluster)
			if err != nil {
				return err
			}
		} else {
			for _, p := range bundle.Spec.Providers {
				if p == cluster.Spec.Provider {
					// install on matching provider
					err := c.createBundlePackageManager(&bundle, cluster)
					if err != nil {
						return err
					}

					err = c.createBundleApplications(&bundle, cluster)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (c *SDSClusterController) createBundlePackageManager(bundle *api.SDSAppBundle, cluster *api.SDSCluster) error {
	// check if bundlePackageManager exists for cluster
	bundlePackageManagerName := bundle.Spec.PackageManager.Name + "-" + cluster.Name

	_, err := c.client.CmaV1alpha1().SDSPackageManagers(viper.GetString(KubernetesNamespaceViperVariableName)).Get(bundlePackageManagerName, v1.GetOptions{})
	if err != nil {
		logger.Infof("the -->%s<-- bundle package manager for cluster -->%s<-- does not exist! creating it", bundle.Name, cluster.Name)

		// create sdsPackageManager
		bundlePackageManager := &api.SDSPackageManager{
			Spec: api.SDSPackageManagerSpec{
				Name:      bundlePackageManagerName,
				Namespace: bundle.Spec.PackageManager.Namespace,
				Version:   bundle.Spec.PackageManager.Version,
				Image:     bundle.Spec.PackageManager.Image,
				ServiceAccount: api.ServiceAccount{
					Name:      bundle.Spec.PackageManager.Name,
					Namespace: bundle.Spec.PackageManager.Namespace,
				},
				Permissions: api.PackageManagerPermissions{
					ClusterWide: bundle.Spec.PackageManager.Permissions.ClusterWide,
					Namespaces:  bundle.Spec.PackageManager.Permissions.Namespaces,
				},
				Cluster: api.SDSClusterRef{
					Name: cluster.Name,
				},
			},
		}

		bundlePackageManager.Name = bundlePackageManagerName
		bundlePackageManager.Namespace = bundle.Spec.Namespace

		// set owner reference
		bundlePackageManager.OwnerReferences = []v1.OwnerReference{
			*v1.NewControllerRef(cluster,
				runtimeSchema.GroupVersionKind{
					Group: api.SchemeGroupVersion.Group,
					Version: api.SchemeGroupVersion.Version,
					Kind: "SDSCluster",
				}),
		}

		newBundlePackageManager, err := c.client.CmaV1alpha1().SDSPackageManagers(viper.GetString(KubernetesNamespaceViperVariableName)).Create(bundlePackageManager)
		if err != nil {
			logger.Errorf("something bad happened when creating -->%s<-- bundle package manager for cluster -->%s<--, error: %s", bundle.Name, cluster.Name, err)
			return err
		}
		logger.Infof("created -->%s<-- bundle package manager for cluster -->%s<--", newBundlePackageManager.Name, cluster.Name)
	}
	return nil
}

func (c *SDSClusterController) createBundleApplications(bundle *api.SDSAppBundle, cluster *api.SDSCluster) error {

	for _, app := range bundle.Spec.Applications {
		sdsApplicationName := app.Name + "-" + app.PackageManager.Name + "-" + cluster.Name

		// check if application exists for this cluster already
		_, err := c.client.CmaV1alpha1().SDSApplications(viper.GetString(KubernetesNamespaceViperVariableName)).Get(sdsApplicationName, v1.GetOptions{})
		if err != nil {
			logger.Infof("the -->%s<-- bundle application -->%s<-- for cluster -->%s<-- does not exist! creating it", bundle.Name, sdsApplicationName, cluster.Name)

			// create sdsApplication
			sdsApplication := &api.SDSApplication{
				Spec: api.SDSApplicationSpec{
					PackageManager: api.SDSPackageManagerRef{
						Name: app.PackageManager.Name,
					},
					Namespace: app.Namespace,
					Name:      sdsApplicationName,
					Chart: api.Chart{
						Name: app.Chart.Name,
						Repository: api.ChartRepository{
							Name: app.Chart.Repository.Name,
							URL:  app.Chart.Repository.URL,
						},
					},
					Values: app.Values,
					Cluster: api.SDSClusterRef{
						Name: cluster.Name,
					},
				},
			}
			sdsApplication.Name = sdsApplicationName
			sdsApplication.Namespace = viper.GetString(KubernetesNamespaceViperVariableName)

			// set owner reference
			sdsApplication.OwnerReferences = []v1.OwnerReference{
				*v1.NewControllerRef(cluster,
					runtimeSchema.GroupVersionKind{
						Group: api.SchemeGroupVersion.Group,
						Version: api.SchemeGroupVersion.Version,
						Kind: "SDSCluster",
					}),
			}

			newSDSApplication, err := c.client.CmaV1alpha1().SDSApplications(viper.GetString(KubernetesNamespaceViperVariableName)).Create(sdsApplication)
			if err != nil {
				logger.Errorf("something bad happened when creating -->%s<-- bundle application -->%s<-- for cluster -->%s<--, error: %s", bundle.Name, sdsApplicationName, cluster.Name, err)
				return err
			}
			logger.Infof("created -->%s<-- bundle application for cluster -->%s<--", newSDSApplication.Name, cluster.Name)
		}
	}

	return nil
}
