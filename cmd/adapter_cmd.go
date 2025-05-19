package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

		clientset, err := getClientSet()

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

		clientset, err := getClientSet()

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
		clientset, err := getClientSet()

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
