package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Manage git worktrees for Gerrit changes",
	Long:  `Setup and cleanup git worktrees for reviewing Gerrit changes in isolation.`,
}

var treeSetupCmd = &cobra.Command{
	Use:   "setup [change-id] [patchset]",
	Short: "Create a worktree for a Gerrit change or new work",
	Long: `Create a git worktree for a specific Gerrit change or for new work.
This allows you to review changes or work on new features in isolation
without affecting your main working directory.

Use --name to create a worktree for new work without a change-id.
Otherwise, provide a change-id to fetch and review an existing change.`,
	Args: cobra.MaximumNArgs(2),
	Run:  runTreeSetup,
}

var treeCleanupCmd = &cobra.Command{
	Use:   "cleanup [change-id|name|path]",
	Short: "Remove worktrees",
	Long: `Remove git worktrees. If no argument is provided, lists all worktrees.
If a change-id is provided, removes the worktree for that change.
If a custom name is provided, removes the worktree with that name.
If a path is provided, removes the worktree at that path.`,
	Args: cobra.MaximumNArgs(1),
	Run:  runTreeCleanup,
}

var treesCmd = &cobra.Command{
	Use:   "trees",
	Short: "List all git worktrees",
	Long:  `List all git worktrees in the current repository.`,
	Run:   runTrees,
}

var (
	worktreeBasePath string
	forceCleanup     bool
	worktreeName     string
)

func init() {
	treeSetupCmd.Flags().StringVarP(&worktreeBasePath, "path", "p", "", "Base path for worktrees (default: ../worktrees)")
	treeSetupCmd.Flags().StringVarP(&worktreeName, "name", "n", "", "Custom name for worktree (for new work without change-id)")
	treeCleanupCmd.Flags().BoolVarP(&forceCleanup, "force", "f", false, "Force cleanup even if worktree has uncommitted changes")
	
	treeCmd.AddCommand(treeSetupCmd)
	treeCmd.AddCommand(treeCleanupCmd)
}

func runTreeSetup(cmd *cobra.Command, args []string) {
	if !isGitRepository() {
		utils.ExitWithError(fmt.Errorf("not in a git repository"))
	}

	// Determine worktree base path
	if worktreeBasePath == "" {
		repoRoot, err := getGitRepoRoot()
		if err != nil {
			utils.ExitWithError(fmt.Errorf("failed to get repository root: %w", err))
		}
		worktreeBasePath = filepath.Join(filepath.Dir(repoRoot), "worktrees")
	}

	// Create worktrees directory if it doesn't exist
	if err := os.MkdirAll(worktreeBasePath, 0755); err != nil {
		utils.ExitWithError(fmt.Errorf("failed to create worktrees directory: %w", err))
	}

	// Handle custom name mode vs change-id mode
	if worktreeName != "" {
		// Custom name mode - create worktree from current HEAD
		if len(args) > 0 {
			utils.ExitWithError(fmt.Errorf("cannot specify change-id when using --name flag"))
		}
		
		worktreePath := filepath.Join(worktreeBasePath, worktreeName)
		
		// Check if worktree already exists
		if _, err := os.Stat(worktreePath); err == nil {
			utils.ExitWithError(fmt.Errorf("worktree already exists at: %s", worktreePath))
		}

		fmt.Printf("Setting up worktree %s from current HEAD...\n", utils.BoldCyan(worktreeName))

		// Create worktree from HEAD
		fmt.Print("Creating worktree... ")
		if err := createWorktree(worktreePath, "HEAD"); err != nil {
			fmt.Println(color.RedString("FAILED"))
			utils.ExitWithError(fmt.Errorf("failed to create worktree: %w", err))
		}
		fmt.Println(color.GreenString("SUCCESS"))

		fmt.Printf("\n%s Worktree created successfully!\n", color.GreenString("✓"))
		fmt.Printf("Path: %s\n", utils.BoldGreen(worktreePath))
		
		// Change to the worktree directory
		if err := os.Chdir(worktreePath); err != nil {
			fmt.Printf("%s Warning: Failed to change to worktree directory: %v\n", color.YellowString("⚠"), err)
		} else {
			fmt.Printf("Changed to worktree directory\n")
		}
		return
	}

	// Change-id mode - need at least one argument
	if len(args) == 0 {
		utils.ExitWithError(fmt.Errorf("must provide change-id or use --name flag for custom worktree"))
	}

	changeID := args[0]
	patchset := ""
	if len(args) > 1 {
		patchset = args[1]
	}

	cfg, err := config.Load()
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to load configuration: %w", err))
	}

	if err := cfg.Validate(); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid configuration: %w", err))
	}

	// Get change details
	change, err := getChangeForFetch(cfg, changeID)
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to get change details: %w", err))
	}

	// Determine patchset number
	patchsetNum := patchset
	if patchsetNum == "" {
		patchsetNum = getCurrentPatchsetNumber(change)
		if patchsetNum == "" {
			utils.ExitWithError(fmt.Errorf("could not determine current patchset"))
		}
	}

	worktreePath := filepath.Join(worktreeBasePath, fmt.Sprintf("change-%s", changeID))

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		utils.ExitWithError(fmt.Errorf("worktree already exists at: %s", worktreePath))
	}

	fmt.Printf("Setting up worktree for change %s (patchset %s)...\n", 
		utils.BoldCyan(changeID), 
		utils.BoldYellow(patchsetNum))

	// Build the refs path
	refsPath := fmt.Sprintf("refs/changes/%s/%s/%s",
		getChangePrefix(changeID),
		changeID,
		patchsetNum)

	// Get git remote URL
	remoteURL := buildRemoteURL(cfg)

	// Fetch the change first
	fmt.Print("Fetching change... ")
	if err := gitFetch(remoteURL, refsPath); err != nil {
		fmt.Println(color.RedString("FAILED"))
		utils.ExitWithError(fmt.Errorf("git fetch failed: %w", err))
	}
	fmt.Println(color.GreenString("SUCCESS"))

	// Create worktree
	fmt.Print("Creating worktree... ")
	if err := createWorktree(worktreePath, "FETCH_HEAD"); err != nil {
		fmt.Println(color.RedString("FAILED"))
		utils.ExitWithError(fmt.Errorf("failed to create worktree: %w", err))
	}
	fmt.Println(color.GreenString("SUCCESS"))

	fmt.Printf("\n%s Worktree created successfully!\n", color.GreenString("✓"))
	fmt.Printf("Path: %s\n", utils.BoldGreen(worktreePath))
	
	// Change to the worktree directory
	if err := os.Chdir(worktreePath); err != nil {
		fmt.Printf("%s Warning: Failed to change to worktree directory: %v\n", color.YellowString("⚠"), err)
	} else {
		fmt.Printf("Changed to worktree directory\n")
	}
}

func runTreeCleanup(cmd *cobra.Command, args []string) {
	if !isGitRepository() {
		utils.ExitWithError(fmt.Errorf("not in a git repository"))
	}

	// If no arguments, list all worktrees
	if len(args) == 0 {
		listWorktrees()
		return
	}

	target := args[0]

	// Check if target is a path, change-id, or custom name
	var worktreePath string
	if strings.HasPrefix(target, "/") || strings.HasPrefix(target, "./") || strings.HasPrefix(target, "../") {
		// Treat as path
		worktreePath = target
	} else {
		// Determine base path for worktrees
		repoRoot, err := getGitRepoRoot()
		if err != nil {
			utils.ExitWithError(fmt.Errorf("failed to get repository root: %w", err))
		}
		worktreeBasePath := filepath.Join(filepath.Dir(repoRoot), "worktrees")
		
		// Try as change-id first (with "change-" prefix), then as custom name
		changeWorktreePath := filepath.Join(worktreeBasePath, fmt.Sprintf("change-%s", target))
		customWorktreePath := filepath.Join(worktreeBasePath, target)
		
		if _, err := os.Stat(changeWorktreePath); err == nil {
			worktreePath = changeWorktreePath
		} else if _, err := os.Stat(customWorktreePath); err == nil {
			worktreePath = customWorktreePath
		} else {
			utils.ExitWithError(fmt.Errorf("worktree not found for '%s' (tried both change-%s and %s)", target, target, target))
		}
	}

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		utils.ExitWithError(fmt.Errorf("worktree does not exist: %s", worktreePath))
	}

	// Check for uncommitted changes unless force is used
	if !forceCleanup {
		if hasUncommittedChanges(worktreePath) {
			utils.ExitWithError(fmt.Errorf("worktree has uncommitted changes. Use --force to cleanup anyway"))
		}
	}

	fmt.Printf("Removing worktree: %s\n", worktreePath)

	// Remove worktree
	if err := removeWorktree(worktreePath); err != nil {
		utils.ExitWithError(fmt.Errorf("failed to remove worktree: %w", err))
	}

	fmt.Printf("%s Worktree removed successfully\n", color.GreenString("✓"))
}

func runTrees(cmd *cobra.Command, args []string) {
	if !isGitRepository() {
		utils.ExitWithError(fmt.Errorf("not in a git repository"))
	}
	
	listWorktrees()
}

func createWorktree(path, commitish string) error {
	cmd := exec.Command("git", "worktree", "add", path, commitish)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func removeWorktree(path string) error {
	cmd := exec.Command("git", "worktree", "remove", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func listWorktrees() {
	cmd := exec.Command("git", "worktree", "list")
	output, err := cmd.Output()
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to list worktrees: %w", err))
	}

	fmt.Println("Current worktrees:")
	fmt.Print(string(output))
}

func hasUncommittedChanges(worktreePath string) bool {
	cmd := exec.Command("git", "-C", worktreePath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

func getGitRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}