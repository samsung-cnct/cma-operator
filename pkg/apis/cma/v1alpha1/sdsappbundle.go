package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppBundlePhase Phase

const (
	AppBundlePhaseNone        = ""
	AppBundlePhasePending     = "Pending"
	AppBundlePhaseInstalling  = "Installing"
	AppBundlePhaseUpgrading   = "Upgrading"
	AppBundlePhaseImplemented = "Implemented"
	AppBundlePhaseFailed      = "Failed"
)

// SDSAppBundleList is a list of sds application bundles.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type SDSAppBundleList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SDSAppBundle `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type SDSAppBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SDSAppBundleSpec   `json:"spec"`
	Status            SDSAppBundleStatus `json:"status"`
}

// SDSAppBundleSpec represents a SDS Application Bundle - a list of applications to be install on clusters automatically after create
// +k8s:openapi-gen=true
type SDSAppBundleSpec struct {
	// What is the AppBundle release's name
	Name string `json:"name"`
	// The Namespace on the management cluster to install the bundle
	Namespace string `json:"namespace"`
	// What Package Manager to create for all applications in this bundle
	PackageManager SDSPackageManagerSpec `json:"sdspackagemanagerspec"`
	// What Applications to create for this bundle
	Applications []SDSApplicationSpec `json:"sdsapplicationspec"`
	// try to install automatically on every cluster
	AutoInstall bool `json:"autoinstall"`
	// providers to install on
	Providers []string `json:"providers"`
	// Kubernetes version and higher to install on
	K8sVersions string `json:"k8sversions"`
}

// The status of an application bundle object
// +k8s:openapi-gen=true
type SDSAppBundleStatus struct {
	Phase      ApplicationPhase `json:"phase"`
	Ready      bool             `json:"ready"`
	Conditions []Condition      `json:"conditions"`
}
