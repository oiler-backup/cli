package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// configCmd is top-level command for actions with configuration.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Display the current configuration",
	Long:  `Display the current configuration loaded from the config file or flags.`,
}

// configSetCmd sets parameters to config.
var configSetCmd = &cobra.Command{
	Use:   "set <parameter>=<value>",
	Short: "Set a configuration parameter",
	Long:  `Set a configuration parameter in the config file.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stopFn := startSpinner("[1/2] Preparing")
		arg := args[0]
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			stopFn()
			log.Fatalf("Invalid argument format. Use <parameter>=<value>")
		}

		parameter := parts[0]
		value := parts[1]

		configPath := filepath.Join(os.Getenv("HOME"), ".oiler", ".config.json")
		configData, err := os.ReadFile(configPath)
		if err != nil {
			stopFn()
			log.Fatalf("Failed to read config file: %v", err)
		}

		if err := json.Unmarshal(configData, &cfg); err != nil {
			stopFn()
			log.Fatalf("Failed to unmarshal config: %v", err)
		}

		switch parameter {
		case "kube-config-path":
			cfg.KubeConfigPath = value
		case "namespace":
			cfg.Namespace = value
		default:
			stopFn()
			log.Fatalf("Unknown parameter: %s", parameter)
		}
		stopFn()

		stopFn = startSpinner("[2/2] Writing result")
		configData, err = json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			stopFn()
			log.Fatalf("Failed to marshal config: %v", err)
		}

		if err := os.WriteFile(configPath, configData, 0644); err != nil {
			stopFn()
			log.Fatalf("Failed to write config file: %v", err)
		}

		stopFn()
		log.Info("Successfully updated config")
	},
}

// configGetCmd shows current configuration.
var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Display the current configuration",
	Long:  `Display the current configuration loaded from the config file or flags.`,
	Run: func(cmd *cobra.Command, args []string) {
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetStyle(table.StyleLight)
		t.AppendHeader(table.Row{"#", "Parameter Name", "Value"})
		t.AppendRow(table.Row{1, "kube_config_path", cfg.KubeConfigPath})
		t.AppendSeparator()
		t.AppendRow(table.Row{2, "namespace", cfg.Namespace})
		t.Render()
	},
}
