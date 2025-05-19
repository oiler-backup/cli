package cmd

import (
	"time"

	"github.com/briandowns/spinner"
	backupv1 "github.com/oiler-backup/core/core/api/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	gvr = schema.GroupVersionResource{
		Group:    backupv1.GroupVersion.Group,
		Version:  backupv1.GroupVersion.Version,
		Resource: "backuprequests",
	}
)

func getConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfigPath)
		if err != nil {
			return nil, err
		}
	}
	return config, nil
}

func getClientSet() (*kubernetes.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func getDynamicClient() (*dynamic.DynamicClient, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return dynClient, nil
}

func startSpinner(text string) func() {
	s := spinner.New([]string{".", "..", "..."}, 500*time.Millisecond)
	s.Prefix = text
	s.Start()

	return s.Stop
}
