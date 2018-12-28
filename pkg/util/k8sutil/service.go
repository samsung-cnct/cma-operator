package k8sutil

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func CreateExternalService(schema corev1.Service, namespace string, config *rest.Config) (bool, error) {
	SetLogger()
	if config == nil {
		config = DefaultConfig
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Errorf("Cannot establish a client connection to kubernetes: %v", err)
		return false, err
	}

	_, err = clientSet.CoreV1().Services(namespace).Create(&schema)
	if err != nil && !IsResourceAlreadyExistsError(err) {
		logger.Infof("Service -->%s<-- in namespace -->%s<-- Cannot be created, error was %v", schema.ObjectMeta.Name, namespace, err)
		return false, err
	} else if IsResourceAlreadyExistsError(err) {
		logger.Infof("Service -->%s<-- in namespace -->%s<-- Already exists, cannot recreate", schema.ObjectMeta.Name, namespace)
		return false, err
	}
	return true, nil
}
