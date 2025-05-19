package k8s

import (
	"fmt"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// GetClient returns Clientset based on kubeconfig file.
func GetClient(kubeConfigPath string) (*kubernetes.Clientset, error) {
	if kubeConfigPath == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeConfigPath = filepath.Join(home, ".kube", "config")
		} else {
			return nil, fmt.Errorf("kubeconfig path not provided and unable to determine home directory")
		}
	}

	// Use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %v", err)
	}

	// Create a new clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %v", err)
	}

	return clientset, nil
}

// GetCustomResourceClient returns client to work with CustomResources.
func GetCustomResourceClient(kubeConfigPath, groupVersion, resource string) (rest.Interface, error) {
	if kubeConfigPath == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeConfigPath = filepath.Join(home, ".kube", "config")
		} else {
			return nil, fmt.Errorf("kubeconfig path not provided and unable to determine home directory")
		}
	}

	// Use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %v", err)
	}

	// Create a REST client
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, fmt.Errorf("error creating REST client: %v", err)
	}

	return restClient, nil
}
