package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	skipPull bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update gerry to the latest version",
	Long: `Update gerry to the latest version by pulling from git and rebuilding.
This command must be run from within the gerry source directory.`,
	Run: runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&skipPull, "skip-pull", false, "Skip git pull and just rebuild")
}

func runUpdate(cmd *cobra.Command, args []string) {
	fmt.Println(color.YellowString("Updating gerry..."))

	// Check if we're in a git repository
	if !isGitRepo() {
		utils.ExitWithError(fmt.Errorf("not in a git repository. Please run this command from the gerry source directory"))
	}

	// Check if Makefile exists
	if !fileExists("Makefile") {
		utils.ExitWithError(fmt.Errorf("Makefile not found. Please run this command from the gerry source directory"))
	}

	if !skipPull {
		// Pull latest changes
		fmt.Print("Pulling latest changes... ")
		if err := runCommand("git", "pull"); err != nil {
			fmt.Println(color.RedString("FAILED"))
			utils.ExitWithError(fmt.Errorf("failed to pull changes: %w", err))
		}
		fmt.Println(color.GreenString("SUCCESS"))
	}

	// Clean and rebuild
	fmt.Print("Cleaning previous build... ")
	if err := runCommand("make", "clean"); err != nil {
		fmt.Println(color.RedString("FAILED"))
		utils.ExitWithError(fmt.Errorf("failed to clean: %w", err))
	}
	fmt.Println(color.GreenString("SUCCESS"))

	fmt.Print("Building gerry... ")
	if err := runCommand("make", "build"); err != nil {
		fmt.Println(color.RedString("FAILED"))
		utils.ExitWithError(fmt.Errorf("failed to build: %w", err))
	}
	fmt.Println(color.GreenString("SUCCESS"))

	// Install the new binary
	fmt.Print("Installing gerry... ")
	if err := installBinary(); err != nil {
		fmt.Println(color.RedString("FAILED"))
		utils.ExitWithError(fmt.Errorf("failed to install: %w", err))
	}
	fmt.Println(color.GreenString("SUCCESS"))

	// Clear shell hash cache to ensure we get the new binary
	fmt.Print("Clearing shell cache... ")
	if err := runCommandQuiet("hash", "-r"); err != nil {
		// hash -r might not exist on all shells, so don't fail
		utils.Debugf("Failed to clear hash cache: %v", err)
	}
	fmt.Println(color.GreenString("SUCCESS"))

	// Simple verification - just check if the binary exists and is executable
	fmt.Print("Verifying installation... ")

	installPath, err := getInstallPath()
	if err != nil {
		fmt.Println(color.YellowString("WARNING"))
		fmt.Printf("Could not determine install path: %v\n", err)
	} else if _, err := os.Stat(installPath); err != nil {
		fmt.Println(color.YellowString("WARNING"))
		fmt.Printf("Binary not found at %s\n", installPath)
	} else {
		fmt.Println(color.GreenString("SUCCESS"))
		fmt.Printf("Binary installed at: %s\n", installPath)
	}

	fmt.Printf("\n%s gerry has been updated successfully!\n", color.GreenString("✓"))
}

func isGitRepo() bool {
	_, err := os.Stat(".git")
	return err == nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installBinary() error {
	binaryPath := "./bin/gerry"

	// Determine install location
	var installPath string

	// Check if user has write access to /usr/local/bin
	if isWritable("/usr/local/bin") {
		installPath = "/usr/local/bin/gerry"
	} else {
		// Fall back to user's bin directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		userBin := filepath.Join(homeDir, "bin")
		if err := os.MkdirAll(userBin, 0755); err != nil {
			return fmt.Errorf("failed to create ~/bin directory: %w", err)
		}

		installPath = filepath.Join(userBin, "gerry")

		// Warn user about PATH
		fmt.Printf("\n%s Installing to ~/bin/gerry. Make sure ~/bin is in your PATH.\n", color.YellowString("⚠"))
		fmt.Println("Add this to your shell profile if needed:")
		fmt.Printf("  %s\n", utils.Cyan("export PATH=\"$HOME/bin:$PATH\""))
	}

	// Copy binary
	if runtime.GOOS == "windows" {
		installPath += ".exe"
		binaryPath += ".exe"
	}

	// Use cp command for copying
	return runCommandQuiet("cp", binaryPath, installPath)
}

func runCommandQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func isWritable(path string) bool {
	// Try to create a temporary file
	testFile := filepath.Join(path, ".gerry-test")
	file, err := os.Create(testFile)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(testFile)
	return true
}

func getInstallPath() (string, error) {
	// Determine where gerry was installed
	if isWritable("/usr/local/bin") {
		return "/usr/local/bin/gerry", nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, "bin", "gerry"), nil
}
