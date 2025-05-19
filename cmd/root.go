package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/oiler-backup/cli/internal/config"
	backupv1 "github.com/oiler-backup/core/core/api/v1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	CM_NAME = "database-config"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "oiler-cli",
	Short: "CLI for Oiler Kubernetes Operator",
	Long:  `CLI tool to interact with Oiler Kubernetes Operator.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Oiler CLI is running")
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Display the current configuration",
	Long:  `Display the current configuration loaded from the config file or flags.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("KubeConfig Path: %s\n", cfg.KubeConfigPath)
		fmt.Printf("Namespace: %s\n", cfg.Namespace)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <parameter>=<value>",
	Short: "Set a configuration parameter",
	Long:  `Set a configuration parameter in the config file.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		arg := args[0]
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			log.Fatalf("Invalid argument format. Use <parameter>=<value>")
		}

		parameter := parts[0]
		value := parts[1]

		configPath := filepath.Join(os.Getenv("HOME"), ".oiler", ".config.json")
		configData, err := os.ReadFile(configPath)
		if err != nil {
			log.Fatalf("Failed to read config file: %v", err)
		}

		if err := json.Unmarshal(configData, &cfg); err != nil {
			log.Fatalf("Failed to unmarshal config: %v", err)
		}

		switch parameter {
		case "kube-config-path":
			cfg.KubeConfigPath = value
		case "namespace":
			cfg.Namespace = value
		default:
			log.Fatalf("Unknown parameter: %s", parameter)
		}

		configData, err = json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal config: %v", err)
		}

		if err := os.WriteFile(configPath, configData, 0644); err != nil {
			log.Fatalf("Failed to write config file: %v", err)
		}

		fmt.Printf("Updated config: %+v\n", cfg)
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Display the current configuration",
	Long:  `Display the current configuration loaded from the config file or flags.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("KubeConfig Path: %s\n", cfg.KubeConfigPath)
		fmt.Printf("Namespace: %s\n", cfg.Namespace)
	},
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage backup resources",
	Long:  `Manage backup resources in the cluster.`,
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all BackupRequest resources",
	Long:  `List all BackupRequest resources in the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := rest.InClusterConfig()
		if err != nil {
			config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfigPath)
			if err != nil {
				log.Fatalf("Failed to create config: %v", err)
			}
		}

		dynClient, err := dynamic.NewForConfig(config)
		if err != nil {
			log.Fatalf("Failed to create dynamic client: %v", err)
		}

		gvr := schema.GroupVersionResource{
			Group:    backupv1.GroupVersion.Group,
			Version:  backupv1.GroupVersion.Version,
			Resource: "backuprequests",
		}

		list, err := dynClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Fatalf("Failed to list BackupRequest resources: %v", err)
		}

		var backupRequests []backupv1.BackupRequest
		for _, item := range list.Items {
			var backupRequest backupv1.BackupRequest
			jsonItem, err := item.MarshalJSON()
			if err != nil {
				log.Fatalf("Failed to unmarshal object: %v", err)
			}
			if err := json.Unmarshal(jsonItem, &backupRequest); err != nil {
				log.Fatalf("Failed to unmarshal BackupRequest resource: %v", err)
			}
			backupRequests = append(backupRequests, backupRequest)
		}
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetStyle(table.StyleLight)
		t.AppendHeader(table.Row{"#", "BackupRequest Name", "Database URI", "Database Name", "Database Type", "Schedule"})
		for i, br := range backupRequests {
			t.AppendRow(table.Row{i + 1, br.Name, br.Spec.DbSpec.URI, br.Spec.DbSpec.DbName, br.Spec.DbSpec.DbType, br.Spec.Schedule})
			t.AppendSeparator()
		}

		t.AppendFooter(table.Row{"", "", "", "", "TOTAL", len(backupRequests)})
		t.Render()
	},
}

var adapterCmd = &cobra.Command{
	Use:   "adapter",
	Short: "Manage adapters",
	Long:  `Manage adapters in the cluster.`,
}

var adapterAddCmd = &cobra.Command{
	Use:   "add <name>=<url>",
	Short: "Add an adapter to the ConfigMap",
	Long:  `Add an adapter to the ConfigMap in the specified namespace.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		arg := args[0]
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			log.Fatalf("Invalid argument format. Use <name>=<url>")
		}

		name := parts[0]
		url := parts[1]

		config, err := rest.InClusterConfig()
		if err != nil {
			config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfigPath)
			if err != nil {
				log.Fatalf("Failed to create config: %v", err)
			}
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatalf("Failed to create clientset: %v", err)
		}

		configMap, err := clientset.CoreV1().ConfigMaps(cfg.Namespace).Get(context.TODO(), CM_NAME, metav1.GetOptions{})
		if err != nil {
			configMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      CM_NAME,
					Namespace: cfg.Namespace,
				},
				Data: map[string]string{
					name: url,
				},
			}

			_, err := clientset.CoreV1().ConfigMaps(cfg.Namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
			if err != nil {
				log.Fatalf("Failed to create ConfigMap: %v", err)
			}

			fmt.Printf("Created ConfigMap '%s' with entry '%s=%s'\n", CM_NAME, name, url)
			return
		}

		configMap.Data[name] = url

		_, err = clientset.CoreV1().ConfigMaps(cfg.Namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
		if err != nil {
			log.Fatalf("Failed to update ConfigMap: %v", err)
		}

		fmt.Printf("Updated ConfigMap '%s' with entry '%s=%s'\n", CM_NAME, name, url)
	},
}

var adapterDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an adapter from the ConfigMap",
	Long:  `Delete an adapter from the ConfigMap in the specified namespace.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		config, err := rest.InClusterConfig()
		if err != nil {
			config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfigPath)
			if err != nil {
				log.Fatalf("Failed to create config: %v", err)
			}
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatalf("Failed to create clientset: %v", err)
		}

		configMap, err := clientset.CoreV1().ConfigMaps(cfg.Namespace).Get(context.TODO(), CM_NAME, metav1.GetOptions{})
		if err != nil {
			log.Fatalf("Failed to get ConfigMap: %v", err)
		}

		if _, exists := configMap.Data[name]; !exists {
			fmt.Printf("Entry '%s' not found in ConfigMap '%s'\n", name, CM_NAME)
			return
		}

		delete(configMap.Data, name)

		_, err = clientset.CoreV1().ConfigMaps(cfg.Namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
		if err != nil {
			log.Fatalf("Failed to update ConfigMap: %v", err)
		}

		fmt.Printf("Deleted entry '%s' from ConfigMap '%s'\n", name, CM_NAME)
	},
}

var adapterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all adapters from the ConfigMap",
	Long:  `List all adapters from the ConfigMap in the specified namespace.`,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := rest.InClusterConfig()
		if err != nil {
			config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfigPath)
			if err != nil {
				log.Fatalf("Failed to create config: %v", err)
			}
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatalf("Failed to create clientset: %v", err)
		}

		configMap, err := clientset.CoreV1().ConfigMaps(cfg.Namespace).Get(context.TODO(), CM_NAME, metav1.GetOptions{})
		if err != nil {
			log.Fatalf("Failed to get ConfigMap: %v", err)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetStyle(table.StyleLight)
		t.AppendHeader(table.Row{"#", "Adapter Name", "Adapter URI"})
		i := 1
		for name, url := range configMap.Data {
			t.AppendRow(table.Row{i, name, url})
			t.AppendSeparator()
			i++
		}
		t.AppendFooter(table.Row{"", "TOTAL", i})

		t.Render()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error while executing command: %v", err)
	}
}

func init() {
	var err error
	cfg, err = config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	backupCmd.AddCommand(backupListCmd)
	adapterCmd.AddCommand(adapterAddCmd)
	adapterCmd.AddCommand(adapterDeleteCmd)
	adapterCmd.AddCommand(adapterListCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(adapterCmd)
}
