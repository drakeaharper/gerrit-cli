package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/drakeaharper/gerrit-cli/internal/utils"
)

type Config struct {
	Server       string `json:"server"`
	Port         int    `json:"port"`
	HTTPPort     int    `json:"http_port,omitempty"`
	User         string `json:"user"`
	HTTPPassword string `json:"http_password,omitempty"`
	SSHKey       string `json:"ssh_key,omitempty"`
	Project      string `json:"project,omitempty"`
}

const (
	configDirName  = ".gerry"
	configFileName = "config.json"
)

var (
	defaultConfig = Config{
		Port: 29418,
	}
)

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, configDirName), nil
}

func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, configFileName), nil
}

func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at %s, run 'gerry init' to create one", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	if config.Port == 0 {
		config.Port = defaultConfig.Port
	}

	// Override with environment variables if set
	if server := os.Getenv("GERRIT_SERVER"); server != "" {
		config.Server = server
	}
	if port := os.Getenv("GERRIT_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &config.Port)
	}
	if user := os.Getenv("GERRIT_USER"); user != "" {
		config.User = user
	}
	if password := os.Getenv("GERRIT_HTTP_PASSWORD"); password != "" {
		config.HTTPPassword = password
	}
	if project := os.Getenv("GERRIT_PROJECT"); project != "" {
		config.Project = project
	}

	return &config, nil
}

func Save(config *Config) error {
	// Validate configuration before saving
	if err := config.Validate(); err != nil {
		return err
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Warning about plain text password storage
	if config.HTTPPassword != "" {
		fmt.Fprintf(os.Stderr, "Warning: HTTP password will be stored in plain text at %s\n", configPath)
		fmt.Fprintf(os.Stderr, "Consider using environment variable GERRIT_HTTP_PASSWORD instead\n")
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (c *Config) Validate() error {
	if err := utils.ValidateServerURL(c.Server); err != nil {
		return fmt.Errorf("invalid server: %w", err)
	}

	if err := utils.ValidateUsername(c.User); err != nil {
		return fmt.Errorf("invalid user: %w", err)
	}

	if err := utils.ValidatePort(c.Port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	if c.HTTPPort != 0 {
		if err := utils.ValidatePort(c.HTTPPort); err != nil {
			return fmt.Errorf("invalid HTTP port: %w", err)
		}
	}

	// Validate SSH key if specified
	if c.SSHKey != "" {
		if err := utils.ValidateSSHKey(c.SSHKey); err != nil {
			return fmt.Errorf("invalid SSH key: %w", err)
		}
	}

	return nil
}

func (c *Config) GetSSHCommand() string {
	sshKey := c.SSHKey
	if sshKey == "" {
		sshKey = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
	}
	return fmt.Sprintf("ssh -p %d -i %s %s@%s gerrit", c.Port, sshKey, c.User, c.Server)
}

func (c *Config) GetRESTURL(path string) string {
	protocol := "https"
	port := c.HTTPPort

	// If no HTTP port specified, try to determine it
	if port == 0 {
		if c.Port == 29418 {
			// Common Gerrit setup: SSH on 29418, HTTPS on 443 or HTTP on 8080
			port = 443
			protocol = "https"
		} else {
			// Use the same port for HTTP
			port = c.Port
		}
	}

	// Determine protocol based on port
	switch port {
	case 80, 8080:
		protocol = "http"
	case 443, 8443:
		protocol = "https"
	default:
		// Default to HTTPS for other ports
		protocol = "https"
	}

	// Don't include port in URL for standard ports
	if (protocol == "https" && port == 443) || (protocol == "http" && port == 80) {
		return fmt.Sprintf("%s://%s/a/%s", protocol, c.Server, path)
	}

	return fmt.Sprintf("%s://%s:%d/a/%s", protocol, c.Server, port, path)
}
