package cmd

import (
	"context"
	"encoding/json"
	"fmt"
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

// restoreCmd is a top-level command for actions with BackupRestores.
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Manage restore resources",
	Long:  `Manage restore resources in the cluster.`,
}

// restoreListCmd lists all BackupRestore instances.
var restoreListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all BackupRestore resources",
	Long:  `List all BackupRestore resources in the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		stopFn := startSpinner("[1/3] Preparing")
		dynClient, err := getDynamicClient()
		if err != nil {
			stopFn()
			log.Fatalf("Failed to get client: %v", err)
		}
		stopFn()

		stopFn = startSpinner("[2/3] Getting BackupRestores")
		list, err := dynClient.Resource(rsGvr).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			stopFn()
			log.Fatalf("Failed to list BackupRestore resources: %v", err)
		}

		var backupRestores []backupv1.BackupRestore
		for _, item := range list.Items {
			var backupRestore backupv1.BackupRestore
			jsonItem, err := item.MarshalJSON()
			if err != nil {
				stopFn()
				log.Fatalf("Failed to unmarshal object: %v", err)
			}
			if err := json.Unmarshal(jsonItem, &backupRestore); err != nil {
				stopFn()
				log.Fatalf("Failed to unmarshal BackupRestore resource: %v", err)
			}
			backupRestores = append(backupRestores, backupRestore)
		}
		stopFn()

		stopFn = startSpinner("[3/3] Generating results")
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetStyle(table.StyleLight)
		t.AppendHeader(table.Row{"#", "BackupRestore Name", "Database URI", "Database Name", "Database Type", "Revision", "Status"})
		for i, br := range backupRestores {
			t.AppendRow(table.Row{i + 1, br.Name, br.Spec.DbSpec.URI, br.Spec.DbSpec.DbName, br.Spec.DbSpec.DbType, br.Spec.BackupRevision, br.Status.Status})
			t.AppendSeparator()
		}
		t.AppendFooter(table.Row{"", "", "", "", "", "TOTAL", len(backupRestores)})
		stopFn()
		t.Render()
	},
}

// restoreCreateCmd creates BackupRestore instance.
var restoreCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a BackupRestore",
	Long:  `Create a BackupRestore`,
	Run: func(cmd *cobra.Command, args []string) {
		stopFn := startSpinner("[1/3] Preparing")
		dbRegex := regexp.MustCompile(`^(?P<dbType>[^@]+)@(?P<dbUri>[^:]+):(?P<dbPort>\d+)/(?P<dbName>.+)$`)
		dbMatches := dbRegex.FindStringSubmatch(db)
		if len(dbMatches) != 5 {
			stopFn()
			log.Fatalf("Invalid --db format. Use dbType@dbUri:dbPort/dbName")
		}
		dbType := dbMatches[1]
		dbUri := dbMatches[2]
		dbPort, err := strconv.Atoi(dbMatches[3])
		if err != nil {
			stopFn()
			log.Fatalf("Port %s is not a valid integer", dbMatches[3])
		}
		dbName := dbMatches[4]

		stopFn()
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
		stopFn = startSpinner("[2/3] Preparing")
		s3Regex := regexp.MustCompile(`^(?P<endpoint>[^/]+)/(?P<bucketName>.+)$`)
		s3Matches := s3Regex.FindStringSubmatch(s3)
		if len(s3Matches) != 3 {
			log.Fatalf("Invalid --s3 format. Use endpoint/bucket")
		}
		s3Endpoint := s3Matches[1]
		s3BucketName := s3Matches[2]

		endpointParts := strings.SplitN(s3Endpoint, "://", 2)
		var protocol, address string
		if len(endpointParts) == 2 {
			protocol = endpointParts[0]
			address = endpointParts[1]
		} else {
			protocol = ""
			address = s3Endpoint
		}

		stopFn()
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
		stopFn = startSpinner("[3/3] Creating BackupRestore")
		backupRestore := backupv1.BackupRestore{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "backup.oiler.backup/v1",
				Kind:       "BackupRestore",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: backupRestoreName,
			},
			Spec: backupv1.BackupRestoreSpec{
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
				BackupRevision: backupRevision,
			},
		}

		dynClient, err := getDynamicClient()
		if err != nil {
			stopFn()
			log.Fatalf("Failed to get client: %v", err)
		}
		unstructuredBackupRestore, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&backupRestore)
		if err != nil {
			stopFn()
			log.Fatalf("Failed to convert BackupRestore to unstructured: %v", err)
		}

		_, err = dynClient.Resource(rsGvr).Create(context.TODO(), &unstructured.Unstructured{Object: unstructuredBackupRestore}, metav1.CreateOptions{})
		if err != nil {
			stopFn()
			log.Fatalf("Failed to create BackupRestore resource: %v", err)
		}
		stopFn()
		log.Infof("Successfully created BackupRestore %s", backupRestoreName)
	},
}

// restoreDeleteCmd deletes BackupRestore.
var restoreDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a BackupRestore",
	Long:  `Delete a BackupRestore`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stopFn := startSpinner("[1/2] Preparing")
		name := args[0]

		dynClient, err := getDynamicClient()
		if err != nil {
			stopFn()
			log.Fatalf("Failed to get client: %v", err)
		}
		stopFn()

		stopFn = startSpinner("[2/2] Deleting BackupRestore")
		err = dynClient.Resource(rsGvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			stopFn()
			log.Fatalf("Failed to delete BackupRestore resource: %v", err)
		}

		stopFn()
		log.Infof("Successfully deleted BackupRestore %s", name)
	},
}

// retoreUpdateCmd updates existing BackupRestore.
var restoreUpdateCmd = &cobra.Command{
	Use:   "update <name> <field>=<value>",
	Short: "Update a field in a BackupRestore",
	Long:  `Update a field in a BackupRestore`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		stopFn := startSpinner("[1/3] Preparing")
		name := args[0]
		fieldValue := args[1]

		parts := strings.SplitN(fieldValue, "=", 2)
		if len(parts) != 2 {
			stopFn()
			log.Fatalf("Invalid argument format. Use <field>=<value>")
		}

		field := parts[0]
		value := parts[1]

		dynClient, err := getDynamicClient()
		if err != nil {
			stopFn()
			log.Fatalf("Failed to get client: %v", err)
		}
		stopFn()

		stopFn = startSpinner("[2/3] Getting BackupRestore")
		backupRestore, err := dynClient.Resource(rsGvr).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			stopFn()
			log.Fatalf("Failed to get BackupRestore resource: %v", err)
		}

		unstructuredBackupRestore := backupRestore.UnstructuredContent()

		fieldParts := strings.Split(field, ".")
		if len(fieldParts) == 0 {
			stopFn()
			log.Fatalf("Invalid field format. Use <field>=<value>")
		}

		stopFn()

		stopFn = startSpinner("[3/3] Updating BackupRestore")
		err = k8s.UpdateField(unstructuredBackupRestore, fieldParts, value)
		if err != nil {
			stopFn()
			log.Fatalf("Failed to update field: %v", err)
		}

		updatedBackupRestore := &unstructured.Unstructured{Object: unstructuredBackupRestore}

		_, err = dynClient.Resource(rsGvr).Update(context.TODO(), updatedBackupRestore, metav1.UpdateOptions{})
		if err != nil {
			stopFn()
			log.Fatalf("Failed to update BackupRestore resource: %v", err)
		}

		stopFn()
		log.Infof("Successfully updated BackupRestore %s", name)
	},
}
