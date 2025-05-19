package cmd

import (
	"fmt"
	"log"

	"github.com/oiler-backup/cli/internal/config"
	"github.com/spf13/cobra"
)

const (
	CM_NAME = "database-config"
)

var cfg *config.Config

// rootCmd is a top-level command
var rootCmd = &cobra.Command{
	Use:   "oiler-cli",
	Short: "CLI for Oiler Kubernetes Operator",
	Long:  `CLI tool to interact with Oiler Kubernetes Operator.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Oiler CLI is running")
	},
}

// Execute executes incoming command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error while executing command: %v", err)
	}
}

// init is a default function to register commands.
func init() {
	var err error
	cfg, err = config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)

	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupDeleteCmd)
	backupCmd.AddCommand(backupUpdateCmd)
	setupFlags()

	adapterCmd.AddCommand(adapterAddCmd)
	adapterCmd.AddCommand(adapterDeleteCmd)
	adapterCmd.AddCommand(adapterListCmd)

	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(adapterCmd)
}
