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

// adapterCmd is a top-level command for actions with adapters ConfigMap.
var adapterCmd = &cobra.Command{
	Use:   "adapter",
	Short: "Manage adapters",
	Long:  `Manage adapters in the cluster.`,
}

// adapterAddCmd adds new adapter to ConfigMap.
var adapterAddCmd = &cobra.Command{
	Use:   "add <name>=<url>",
	Short: "Add an adapter to the ConfigMap",
	Long:  `Add an adapter to the ConfigMap in the specified namespace.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stopFn := startSpinner("[1/3] Preparing")
		arg := args[0]
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			stopFn()
			log.Fatalf("Invalid argument format. Use <name>=<url>")
		}

		name := parts[0]
		url := parts[1]

		clientset, err := getClientSet()

		stopFn()

		stopFn = startSpinner("[2/3] Getting config map")
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
				stopFn()
				log.Fatalf("Failed to create ConfigMap: %v", err)
			}
			stopFn()
			fmt.Printf("Created ConfigMap '%s' with entry '%s=%s'\n", CM_NAME, name, url)
			return
		}
		stopFn()

		stopFn = startSpinner("[3/3] Updating existing config map")
		configMap.Data[name] = url

		_, err = clientset.CoreV1().ConfigMaps(cfg.Namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
		if err != nil {
			stopFn()
			log.Fatalf("Failed to update ConfigMap: %v", err)
		}
		stopFn()
		fmt.Printf("Updated ConfigMap '%s' with entry '%s=%s'\n", CM_NAME, name, url)
	},
}

// adapterDeleteCmd deletes existing adapter from ConfigMap.
var adapterDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an adapter from the ConfigMap",
	Long:  `Delete an adapter from the ConfigMap in the specified namespace.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		stopFn := startSpinner("[1/3] Preparing")
		clientset, err := getClientSet()

		stopFn()
		stopFn = startSpinner("[2/3] Getting config map")
		configMap, err := clientset.CoreV1().ConfigMaps(cfg.Namespace).Get(context.TODO(), CM_NAME, metav1.GetOptions{})
		if err != nil {
			stopFn()
			log.Fatalf("Failed to get ConfigMap: %v", err)
		}

		if _, exists := configMap.Data[name]; !exists {
			stopFn()
			fmt.Printf("Entry '%s' not found in ConfigMap '%s'\n", name, CM_NAME)
			return
		}
		stopFn()

		stopFn = startSpinner("[3/3] Updating config map")
		delete(configMap.Data, name)

		_, err = clientset.CoreV1().ConfigMaps(cfg.Namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
		if err != nil {
			stopFn()
			log.Fatalf("Failed to update ConfigMap: %v", err)
		}

		stopFn()
		fmt.Printf("Deleted entry '%s' from ConfigMap '%s'\n", name, CM_NAME)
	},
}

// adapterListCmd lists all active adapters.
var adapterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all adapters from the ConfigMap",
	Long:  `List all adapters from the ConfigMap in the specified namespace.`,
	Run: func(cmd *cobra.Command, args []string) {
		stopFn := startSpinner("[1/3] Preparing")
		clientset, err := getClientSet()
		stopFn()

		stopFn = startSpinner("[2/3] Getting config map")
		configMap, err := clientset.CoreV1().ConfigMaps(cfg.Namespace).Get(context.TODO(), CM_NAME, metav1.GetOptions{})
		if err != nil {
			stopFn()
			log.Fatalf("Failed to get ConfigMap: %v", err)
		}
		stopFn()

		stopFn = startSpinner("[3/3] Generating results")
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
		t.AppendFooter(table.Row{"", "TOTAL", i - 1})

		stopFn()
		t.Render()
	},
}
