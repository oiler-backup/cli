package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/oiler-backup/cli/internal/k8s"
	backupv1 "github.com/oiler-backup/core/core/api/v1"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

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
		dynClient, err := getDynamicClient()

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

		dynClient, err := getDynamicClient()

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

		dynClient, err := getDynamicClient()

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

		dynClient, err := getDynamicClient()

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
