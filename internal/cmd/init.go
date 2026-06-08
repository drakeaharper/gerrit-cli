package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gerry configuration",
	Long:  `Interactive setup wizard to configure your Gerrit connection.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println(color.YellowString("Welcome to gerry setup wizard!"))
	fmt.Println("This will guide you through setting up your Gerrit connection.")

	cfg := &config.Config{}

	// Detect an existing configuration and pre-fill prompts with its values.
	existing, _ := config.Load()
	if existing != nil {
		*cfg = *existing
		if configPath, err := config.GetConfigPath(); err == nil {
			fmt.Printf("\nExisting configuration detected at %s.\n", configPath)
		}
		fmt.Println("Press Enter to keep the current value shown as the default.")
	}

	// Default for the SSH port prompt
	sshPortDefault := "29418"
	if cfg.Port != 0 {
		sshPortDefault = fmt.Sprintf("%d", cfg.Port)
	}
	// Default for the username prompt
	userDefault := os.Getenv("USER")
	if cfg.User != "" {
		userDefault = cfg.User
	}

	// Server details
	serverPrompt := &survey.Input{
		Message: "Gerrit server hostname:",
		Default: cfg.Server,
		Help:    "Example: gerrit.example.com",
	}
	if err := survey.AskOne(serverPrompt, &cfg.Server, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// Port
	portPrompt := &survey.Input{
		Message: "SSH port:",
		Default: sshPortDefault,
		Help:    "Default Gerrit SSH port is 29418",
	}
	var portStr string
	if err := survey.AskOne(portPrompt, &portStr); err != nil {
		return err
	}
	fmt.Sscanf(portStr, "%d", &cfg.Port)

	// Username
	userPrompt := &survey.Input{
		Message: "Your Gerrit username:",
		Default: userDefault,
	}
	if err := survey.AskOne(userPrompt, &cfg.User, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// Inform user about SSH key handling
	fmt.Println("\nNote: SSH key selection will be handled by your SSH client configuration (~/.ssh/config)")
	fmt.Println("Make sure your SSH keys are set up correctly for", cfg.Server)

	// Test SSH connection
	fmt.Print("\nTesting SSH connection... ")
	sshClient := gerrit.NewSSHClient(cfg)
	if err := sshClient.TestConnection(); err != nil {
		fmt.Println(color.RedString("FAILED"))
		fmt.Printf("Error: %v\n", err)
		return fmt.Errorf("SSH connection failed, check your SSH configuration")
	}
	fmt.Println(color.GreenString("SUCCESS"))

	// HTTP Password
	hadPassword := cfg.HTTPPassword != ""
	httpPortDefault := ""
	if cfg.HTTPPort != 0 {
		httpPortDefault = fmt.Sprintf("%d", cfg.HTTPPort)
	}

	useREST := false
	restPrompt := &survey.Confirm{
		Message: "Do you want to configure REST API access? (recommended for full functionality)",
		Default: true,
	}
	if err := survey.AskOne(restPrompt, &useREST); err != nil {
		return err
	}

	if useREST {
		// Ask for HTTP port
		httpPortPrompt := &survey.Input{
			Message: "HTTP/HTTPS port (optional, blank = use server default):",
			Default: httpPortDefault,
			Help:    "Only set this if your Gerrit uses a non-standard port. Common ports: 443 (HTTPS), 8080 (HTTP), 8443 (HTTPS alternate)",
		}
		var httpPortStr string
		if err := survey.AskOne(httpPortPrompt, &httpPortStr); err != nil {
			return err
		}
		if httpPortStr != "" {
			fmt.Sscanf(httpPortStr, "%d", &cfg.HTTPPort)
		}

		passwordMessage := "HTTP password:"
		if hadPassword {
			passwordMessage = "HTTP password (leave blank to keep existing):"
		}
		httpPasswordPrompt := &survey.Password{
			Message: passwordMessage,
			Help:    "Found in Gerrit Settings → HTTP Password",
		}
		var httpPassword string
		if err := survey.AskOne(httpPasswordPrompt, &httpPassword); err != nil {
			return err
		}
		// Keep the existing password when the user leaves the prompt blank.
		if httpPassword != "" {
			cfg.HTTPPassword = httpPassword
		}

		// Test REST connection
		fmt.Print("Testing REST API connection... ")
		restClient := gerrit.NewRESTClient(cfg)
		if err := restClient.TestConnection(); err != nil {
			fmt.Println(color.RedString("FAILED"))
			fmt.Printf("Error: %v\n", err)

			switch {
			case strings.Contains(err.Error(), "401"):
				// Auth failure: the HTTP password is missing, wrong, or not generated yet
				fmt.Printf("\nThis is an authentication problem, not a port problem.\n")
				fmt.Printf("Generate an HTTP password in Gerrit: Settings → HTTP Credentials\n")
				fmt.Printf("  %s\n", cfg.GetHTTPBaseURL()+"/settings/#HTTPCredentials")
				fmt.Println("Then run 'gerry init' again and paste the generated password.")
			case cfg.HTTPPort == 0:
				// No explicit port and not an auth error: a non-standard port may be needed
				fmt.Println("\nThe server default port did not work. Common HTTP ports are:")
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
		return err
	}

	// Save configuration
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("\n%s Configuration saved to: %s\n", color.GreenString("✓"), configPath)
	fmt.Println("\nYou're all set! Try running 'gerry list' to see your open changes.")
	return nil
}

func init() {
	// Remove unnecessary flags for init command
	initCmd.Flags().BoolP("force", "f", false, "Force overwrite existing configuration")
}
