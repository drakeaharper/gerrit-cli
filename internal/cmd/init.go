package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gerry configuration",
	Long:  `Interactive setup wizard to configure your Gerrit connection.`,
	Run:   runInit,
}

func runInit(cmd *cobra.Command, args []string) {
	fmt.Println(color.YellowString("Welcome to gerry setup wizard!"))
	fmt.Println("This will guide you through setting up your Gerrit connection.")

	cfg := &config.Config{}

	// Server details
	serverPrompt := &survey.Input{
		Message: "Gerrit server hostname:",
		Help:    "Example: gerrit.example.com",
	}
	if err := survey.AskOne(serverPrompt, &cfg.Server, survey.WithValidator(survey.Required)); err != nil {
		utils.ExitWithError(err)
	}

	// Port
	portPrompt := &survey.Input{
		Message: "SSH port:",
		Default: "29418",
		Help:    "Default Gerrit SSH port is 29418",
	}
	var portStr string
	if err := survey.AskOne(portPrompt, &portStr); err != nil {
		utils.ExitWithError(err)
	}
	fmt.Sscanf(portStr, "%d", &cfg.Port)

	// Username
	userPrompt := &survey.Input{
		Message: "Your Gerrit username:",
		Default: os.Getenv("USER"),
	}
	if err := survey.AskOne(userPrompt, &cfg.User, survey.WithValidator(survey.Required)); err != nil {
		utils.ExitWithError(err)
	}

	// SSH Key
	defaultSSHKey := filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
	sshKeyPrompt := &survey.Input{
		Message: "Path to SSH private key:",
		Default: defaultSSHKey,
		Help:    "Path to your SSH private key for Gerrit authentication",
	}
	if err := survey.AskOne(sshKeyPrompt, &cfg.SSHKey); err != nil {
		utils.ExitWithError(err)
	}

	// Test SSH connection
	fmt.Print("\nTesting SSH connection... ")
	sshClient := gerrit.NewSSHClient(cfg)
	if err := sshClient.TestConnection(); err != nil {
		fmt.Println(color.RedString("FAILED"))
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nPlease check your SSH configuration and try again.")
		os.Exit(1)
	}
	fmt.Println(color.GreenString("SUCCESS"))

	// HTTP Password
	useREST := false
	restPrompt := &survey.Confirm{
		Message: "Do you want to configure REST API access? (recommended for full functionality)",
		Default: true,
	}
	if err := survey.AskOne(restPrompt, &useREST); err != nil {
		utils.ExitWithError(err)
	}

	if useREST {
		// Ask for HTTP port
		httpPortPrompt := &survey.Input{
			Message: "HTTP/HTTPS port (leave blank to auto-detect):",
			Help:    "Common ports: 443 (HTTPS), 8080 (HTTP), 8443 (HTTPS alternate)",
		}
		var httpPortStr string
		if err := survey.AskOne(httpPortPrompt, &httpPortStr); err != nil {
			utils.ExitWithError(err)
		}
		if httpPortStr != "" {
			fmt.Sscanf(httpPortStr, "%d", &cfg.HTTPPort)
		}

		httpPasswordPrompt := &survey.Password{
			Message: "HTTP password:",
			Help:    "Found in Gerrit Settings → HTTP Password",
		}
		if err := survey.AskOne(httpPasswordPrompt, &cfg.HTTPPassword); err != nil {
			utils.ExitWithError(err)
		}

		// Test REST connection
		fmt.Print("Testing REST API connection... ")
		restClient := gerrit.NewRESTClient(cfg)
		if err := restClient.TestConnection(); err != nil {
			fmt.Println(color.RedString("FAILED"))
			fmt.Printf("Error: %v\n", err)

			// If auto-detect failed, suggest trying with explicit port
			if cfg.HTTPPort == 0 {
				fmt.Println("\nAuto-detection may have failed. Common HTTP ports are:")
				fmt.Println("  - 443 (HTTPS)")
				fmt.Println("  - 8080 (HTTP)")
				fmt.Println("  - 8443 (HTTPS alternate)")
				fmt.Println("\nYou can run 'gerry init' again to specify the HTTP port explicitly.")
			}

			fmt.Println("\nREST API access will be disabled. You can update the configuration later.")
			cfg.HTTPPassword = ""
			cfg.HTTPPort = 0
		} else {
			fmt.Println(color.GreenString("SUCCESS"))
		}
	}

	// Default project (optional)
	projectPrompt := &survey.Input{
		Message: "Default project (optional):",
		Help:    "You can specify a default project to use with commands",
	}
	if err := survey.AskOne(projectPrompt, &cfg.Project); err != nil {
		utils.ExitWithError(err)
	}

	// Save configuration
	if err := config.Save(cfg); err != nil {
		utils.ExitWithError(fmt.Errorf("failed to save configuration: %w", err))
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("\n%s Configuration saved to: %s\n", color.GreenString("✓"), configPath)
	fmt.Println("\nYou're all set! Try running 'gerry list' to see your open changes.")
}

func init() {
	// Remove unnecessary flags for init command
	initCmd.Flags().BoolP("force", "f", false, "Force overwrite existing configuration")
}
