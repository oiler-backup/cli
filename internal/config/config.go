package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// A Config stores configuration.
type Config struct {
	KubeConfigPath string `mapstructure:"kube_config_path" json:"kube_config_path"`
	Namespace      string `mapstructure:"namespace" json:"namespace"`
}

// LoadConfig reads configuration file and fills Config up.
func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configPath := filepath.Join(home, ".oiler", ".config.json")

	viper.AddConfigPath(filepath.Dir(configPath))
	viper.SetConfigName(filepath.Base(configPath[:len(configPath)-len(filepath.Ext(configPath))]))
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
