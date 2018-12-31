package k8sutil

import (
	"github.com/samsung-cnct/cma-operator/pkg/apis/cma/v1alpha1"
	api "github.com/samsung-cnct/cma-operator/pkg/apis/cma/v1alpha1"
	"github.com/spf13/viper"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeSchema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GenerateIngress(name string, cluster string, service string) v1beta1.Ingress {

	annotations := map[string]string{
		"kubernetes.io/ingress.class":                  "nginx",
		"nginx.ingress.kubernetes.io/backend-protocol": "HTTPS",
		"nginx.ingress.kubernetes.io/ssl-redirect":     "true",
		"nginx.ingress.kubernetes.io/rewrite-target":   "/",
	}

	return v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{
				{
					SecretName: "cma-api-proxy-tls",
					Hosts: []string{
						viper.GetString("cma-api-proxy"),
					},
				},
			},
			Rules: []v1beta1.IngressRule{
				{
					Host: viper.GetString("cma-api-proxy"),
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/" + cluster,
									Backend: v1beta1.IngressBackend{
										ServiceName: service,
										ServicePort: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 443,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func CreateIngress(schema v1beta1.Ingress, sdsCluster *v1alpha1.SDSCluster, config *rest.Config) (bool, error) {
	SetLogger()
	if config == nil {
		config = DefaultConfig
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Errorf("Cannot establish a client connection to kubernetes: %v", err)
		return false, err
	}

	if sdsCluster.Spec.Provider == "azure" {
		backendService, err := clientSet.CoreV1().Services(sdsCluster.GetNamespace()).Get(
			schema.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.ServiceName,
			metav1.GetOptions{})
		if err != nil {
			logger.Infof("Could not get backend service for ingress -->%s<-- in namespace -->%s<--, error was %v",
				schema.ObjectMeta.Name, sdsCluster.GetNamespace(), err)
			return false, err
		}

		schema.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/configuration-snippet"] = "set $best_http_host \"" +
			backendService.Spec.ExternalName + "\";\n" +
			"set $pass_access_scheme \"https\";\n" +
			"proxy_set_header Host $best_http_host;\n" +
			"proxy_set_header X-Forwarded-Proto $scheme;"
	}

	// set owner reference
	schema.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(sdsCluster,
			runtimeSchema.GroupVersionKind{
				Group:   api.SchemeGroupVersion.Group,
				Version: api.SchemeGroupVersion.Version,
				Kind:    "SDSCluster",
			}),
	}

	_, err = clientSet.ExtensionsV1beta1().Ingresses(sdsCluster.GetNamespace()).Create(&schema)
	if err != nil && !IsResourceAlreadyExistsError(err) {
		logger.Infof("Ingress -->%s<-- in namespace -->%s<-- Cannot be created, error was %v",
			schema.ObjectMeta.Name, sdsCluster.GetNamespace(), err)
		return false, err
	} else if IsResourceAlreadyExistsError(err) {
		logger.Infof("Ingress -->%s<-- in namespace -->%s<-- Already exists, cannot recreate",
			schema.ObjectMeta.Name, sdsCluster.GetNamespace())
		return false, err
	}
	return true, nil
}
