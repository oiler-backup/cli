package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

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
