package k8sutil

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetSecret(name string, namespace string, config *rest.Config) (corev1.Secret, error) {
	SetLogger()
	if config == nil {
		config = DefaultConfig
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Errorf("Cannot establish a client connection to kubernetes: %v", err)
		return corev1.Secret{}, err
	}

	secretResult, err := clientSet.CoreV1().Secrets(namespace).Get(name, v1.GetOptions{})
	if err != nil {
		return corev1.Secret{}, err
	}
	return *secretResult, nil
}

func CreateSecret(name string, namespace string, dataKeyName string, secretData []byte, config *rest.Config) (error) {
	SetLogger()
	if config == nil {
		config = DefaultConfig
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Errorf("Cannot establish a client connection to kubernetes: %v", err)
		return err
	}

	dataMap := make(map[string][]byte)
	dataMap[dataKeyName] = secretData

	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{Name: name},
		Type:       corev1.SecretTypeOpaque,
		Data:       dataMap,
	}

	_, err = clientSet.CoreV1().Secrets(namespace).Create(secret)
	return nil
}

func DeleteSecret(name string, namespace string, config *rest.Config) (error) {
	SetLogger()
	if config == nil {
		config = DefaultConfig
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Errorf("Cannot establish a client connection to kubernetes: %v", err)
		return err
	}

	err = clientSet.CoreV1().Secrets(namespace).Delete(name, &v1.DeleteOptions{})
	return nil
}
