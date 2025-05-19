package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/oiler-backup/cli/internal/config"
	"github.com/oiler-backup/cli/internal/k8s"
	backupv1 "github.com/oiler-backup/core/core/api/v1"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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
		t.AppendHeader(table.Row{"#", "BackupRequest Name", "Database URI", "Database Name", "Database Type", "Schedule", "Status"})
		for i, br := range backupRequests {
			t.AppendRow(table.Row{i + 1, br.Name, br.Spec.DbSpec.URI, br.Spec.DbSpec.DbName, br.Spec.DbSpec.DbType, br.Spec.Schedule, br.Status.Status})
			t.AppendSeparator()
		}

		t.AppendFooter(table.Row{"", "", "", "", "", "TOTAL", len(backupRequests)})
		t.Render()
	},
}

var (
	db                string
	dbUser            string
	dbPass            string
	dbUserStdin       bool
	dbPassStdin       bool
	s3                string
	s3AccessKey       string
	s3SecretKey       string
	s3AccessKeyStdin  bool
	s3SecretKeyStdin  bool
	schedule          string
	maxBackupCount    int64
	backupRequestName string
)

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a BackupRequest",
	Long:  `Create a BackupRequest in the specified namespace.`,
	Run: func(cmd *cobra.Command, args []string) {
		dbRegex := regexp.MustCompile(`^(?P<dbType>[^@]+)@(?P<dbUri>[^:]+):(?P<dbPort>\d+)/(?P<dbName>.+)$`)
		dbMatches := dbRegex.FindStringSubmatch(db)
		if len(dbMatches) != 5 {
			log.Fatalf("Invalid --db format. Use dbType@dbUri:dbPort/dbName")
		}
		dbType := dbMatches[1]
		dbUri := dbMatches[2]
		dbPort, err := strconv.Atoi(dbMatches[3])
		if err != nil {
			log.Fatalf("Port %s is not a valid integer", dbMatches[3])
		}
		dbName := dbMatches[4]

		var dbUserInput, dbPassInput string
		if dbUserStdin {
			fmt.Print("Enter DB User: ")
			byteUser, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				log.Fatalf("Failed to read DB User: %v", err)
			}
			dbUserInput = string(byteUser)
			fmt.Println()
		} else {
			dbUserInput = dbUser
		}

		if dbPassStdin {
			fmt.Print("Enter DB Password: ")
			bytePass, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				log.Fatalf("Failed to read DB Password: %v", err)
			}
			dbPassInput = string(bytePass)
			fmt.Println()
		} else {
			dbPassInput = dbPass
		}

		s3Regex := regexp.MustCompile(`^(?P<endpoint>[^/]+)/(?P<bucketName>.+)$`)
		s3Matches := s3Regex.FindStringSubmatch(s3)
		if len(s3Matches) != 3 {
			log.Fatalf("Invalid --s3 format. Use endpoint/bucket")
		}
		s3Endpoint := s3Matches[1]
		s3BucketName := s3Matches[2]

		// Разделяем endpoint на протокол и адрес
		endpointParts := strings.SplitN(s3Endpoint, "://", 2)
		var protocol, address string
		if len(endpointParts) == 2 {
			protocol = endpointParts[0]
			address = endpointParts[1]
		} else {
			protocol = ""
			address = s3Endpoint
		}

		var s3AccessKeyInput, s3SecretKeyInput string
		if s3AccessKeyStdin {
			fmt.Print("Enter S3 Access Key: ")
			byteAccessKey, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				log.Fatalf("Failed to read S3 Access Key: %v", err)
			}
			s3AccessKeyInput = string(byteAccessKey)
			fmt.Println()
		} else {
			s3AccessKeyInput = s3AccessKey
		}

		if s3SecretKeyStdin {
			fmt.Print("Enter S3 Secret Key: ")
			byteSecretKey, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				log.Fatalf("Failed to read S3 Secret Key: %v", err)
			}
			s3SecretKeyInput = string(byteSecretKey)
			fmt.Println()
		} else {
			s3SecretKeyInput = s3SecretKey
		}

		backupRequest := backupv1.BackupRequest{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "backup.oiler.backup/v1",
				Kind:       "BackupRequest",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: backupRequestName,
			},
			Spec: backupv1.BackupRequestSpec{
				DbSpec: backupv1.DatabaseSpec{
					DbType: dbType,
					URI:    dbUri,
					Port:   dbPort,
					User:   dbUserInput,
					Pass:   dbPassInput,
					DbName: dbName,
				},
				S3Spec: backupv1.S3Spec{
					Endpoint:   fmt.Sprintf("%s://%s", protocol, address),
					BucketName: s3BucketName,
					Auth: backupv1.S3Auth{
						AccessKey: s3AccessKeyInput,
						SecretKey: s3SecretKeyInput,
					},
				},
				Schedule:       schedule,
				MaxBackupCount: maxBackupCount,
			},
		}

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

		unstructuredBackupRequest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&backupRequest)
		if err != nil {
			log.Fatalf("Failed to convert BackupRequest to unstructured: %v", err)
		}

		_, err = dynClient.Resource(gvr).Create(context.TODO(), &unstructured.Unstructured{Object: unstructuredBackupRequest}, metav1.CreateOptions{})
		if err != nil {
			log.Fatalf("Failed to create BackupRequest resource: %v", err)
		}

		fmt.Printf("BackupRequest '%s' created successfully\n", backupRequestName)
	},
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a BackupRequest",
	Long:  `Delete a BackupRequest in the specified namespace.`,
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

		dynClient, err := dynamic.NewForConfig(config)
		if err != nil {
			log.Fatalf("Failed to create dynamic client: %v", err)
		}

		gvr := schema.GroupVersionResource{
			Group:    backupv1.GroupVersion.Group,
			Version:  backupv1.GroupVersion.Version,
			Resource: "backuprequests",
		}

		err = dynClient.Resource(gvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			log.Fatalf("Failed to delete BackupRequest resource: %v", err)
		}

		fmt.Printf("BackupRequest '%s' deleted successfully\n", name)
	},
}

var backupUpdateCmd = &cobra.Command{
	Use:   "update <name> <field>=<value>",
	Short: "Update a field in a BackupRequest",
	Long:  `Update a field in a BackupRequest in the specified namespace.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		fieldValue := args[1]

		parts := strings.SplitN(fieldValue, "=", 2)
		if len(parts) != 2 {
			log.Fatalf("Invalid argument format. Use <field>=<value>")
		}

		field := parts[0]
		value := parts[1]

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

		backupRequest, err := dynClient.Resource(gvr).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			log.Fatalf("Failed to get BackupRequest resource: %v", err)
		}

		unstructuredBackupRequest := backupRequest.UnstructuredContent()

		fieldParts := strings.Split(field, ".")
		if len(fieldParts) == 0 {
			log.Fatalf("Invalid field format. Use <field>=<value>")
		}

		err = k8s.UpdateField(unstructuredBackupRequest, fieldParts, value)
		if err != nil {
			log.Fatalf("Failed to update field: %v", err)
		}

		updatedBackupRequest := &unstructured.Unstructured{Object: unstructuredBackupRequest}

		_, err = dynClient.Resource(gvr).Update(context.TODO(), updatedBackupRequest, metav1.UpdateOptions{})
		if err != nil {
			log.Fatalf("Failed to update BackupRequest resource: %v", err)
		}

		fmt.Printf("BackupRequest '%s' updated successfully\n", name)
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
	backupCreateCmd.Flags().StringVar(&db, "db", "", "DB specification in the format dbType@dbUri:dbPort/dbName")
	backupCreateCmd.Flags().StringVar(&dbUser, "db-user", "", "DB user")
	backupCreateCmd.Flags().StringVar(&dbPass, "db-pass", "", "DB password")
	backupCreateCmd.Flags().BoolVar(&dbUserStdin, "db-user-stdin", false, "Prompt for DB user from stdin")
	backupCreateCmd.Flags().BoolVar(&dbPassStdin, "db-pass-stdin", false, "Prompt for DB password from stdin")
	backupCreateCmd.Flags().StringVar(&s3, "s3", "", "S3 specification in the format endpoint:port/bucket")
	backupCreateCmd.Flags().StringVar(&s3AccessKey, "s3-access-key", "", "S3 access key")
	backupCreateCmd.Flags().StringVar(&s3SecretKey, "s3-secret-key", "", "S3 secret key")
	backupCreateCmd.Flags().BoolVar(&s3AccessKeyStdin, "s3-access-key-stdin", false, "Prompt for S3 access key from stdin")
	backupCreateCmd.Flags().BoolVar(&s3SecretKeyStdin, "s3-secret-key-stdin", false, "Prompt for S3 secret key from stdin")
	backupCreateCmd.Flags().StringVar(&schedule, "schedule", "*/1 * * * *", "Cron schedule for backups")
	backupCreateCmd.Flags().Int64Var(&maxBackupCount, "max-backup-count", 2, "Maximum number of backups to retain")
	backupCreateCmd.Flags().StringVar(&backupRequestName, "name", "", "Name of the BackupRequest")
	backupCreateCmd.MarkFlagRequired("name")
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupDeleteCmd)
	backupCmd.AddCommand(backupUpdateCmd)
	adapterCmd.AddCommand(adapterAddCmd)
	adapterCmd.AddCommand(adapterDeleteCmd)
	adapterCmd.AddCommand(adapterListCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(adapterCmd)
}
