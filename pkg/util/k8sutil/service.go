package k8sutil

import (
	"github.com/samsung-cnct/cma-operator/pkg/apis/cma/v1alpha1"
	api "github.com/samsung-cnct/cma-operator/pkg/apis/cma/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeSchema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GenerateExternalService(name string, externalName string) corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},

		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: externalName,
			Ports: []corev1.ServicePort{
				{
					Port: 443,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 443,
					},
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
}

func CreateExternalService(schema corev1.Service, sdsCluster *v1alpha1.SDSCluster, config *rest.Config) (bool, error) {
	SetLogger()
	if config == nil {
		config = DefaultConfig
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Errorf("Cannot establish a client connection to kubernetes: %v", err)
		return false, err
	}

	schema.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(sdsCluster,
			runtimeSchema.GroupVersionKind{
				Group:   api.SchemeGroupVersion.Group,
				Version: api.SchemeGroupVersion.Version,
				Kind:    "SDSCluster",
			}),
	}

	_, err = clientSet.CoreV1().Services(sdsCluster.GetNamespace()).Create(&schema)
	if err != nil && !IsResourceAlreadyExistsError(err) {
		logger.Infof("Service -->%s<-- in namespace -->%s<-- Cannot be created, error was %v",
			schema.ObjectMeta.Name, sdsCluster.GetNamespace(), err)
		return false, err
	} else if IsResourceAlreadyExistsError(err) {
		logger.Infof("Service -->%s<-- in namespace -->%s<-- Already exists, cannot recreate",
			schema.ObjectMeta.Name, sdsCluster.GetNamespace())
		return false, err
	}
	return true, nil
}
