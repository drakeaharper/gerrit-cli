package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	noCommit           bool
	cherryPickNoVerify bool
)

var cherryPickCmd = &cobra.Command{
	Use:     "cherry <change-id> [patchset]",
	Aliases: []string{"cherry-pick"},
	Short:   "Cherry-pick a change",
	Long:    `Fetch and cherry-pick a change. If patchset is not specified, uses the current patch set.`,
	Args:    cobra.RangeArgs(1, 2),
	RunE: runCherryPick,
}

func init() {
	cherryPickCmd.Flags().BoolVarP(&noCommit, "no-commit", "n", false, "Don't commit the cherry-pick")
	cherryPickCmd.Flags().BoolVar(&cherryPickNoVerify, "no-verify", false, "Skip git hooks during cherry-pick")
}

func runCherryPick(cmd *cobra.Command, args []string) error {
	changeID := args[0]
	// Validate change ID
	if err := utils.ValidateChangeID(changeID); err != nil {
		return fmt.Errorf("invalid change ID: %w", err)
	}

	patchset := ""
	if len(args) > 1 {
		patchset = args[1]
		// Basic validation for patchset number
		if patchset != "" && !regexp.MustCompile(`^\d+$`).MatchString(patchset) {
			return fmt.Errorf("invalid patchset number: %s", patchset)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Check if we're in a git repository
	if !isGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	// Check if working directory is clean
	if !isWorkingDirectoryClean() {
		return fmt.Errorf("working directory is not clean. Please commit or stash your changes")
	}

	utils.Debugf("Cherry-picking change %s patchset %s", changeID, patchset)

	// Get change details to build the fetch ref
	change, err := getChangeForFetch(cfg, changeID)
	if err != nil {
		return fmt.Errorf("failed to get change details: %w", err)
	}

	subject := change.Subject

	// Determine patchset number
	patchsetNum := patchset
	if patchsetNum == "" {
		patchsetNum = getCurrentPatchsetNumber(change)
		if patchsetNum == "" {
			return fmt.Errorf("could not determine current patchset")
		}
	}

	// Build the refs path
	refsPath := fmt.Sprintf("refs/changes/%s/%s/%s",
		getChangePrefix(changeID),
		changeID,
		patchsetNum)

	// Get git remote URL for the server
	remoteURL := buildRemoteURL(cfg)

	fmt.Printf("Cherry-picking change %s (patchset %s) from %s...\n",
		utils.BoldCyan(changeID),
		utils.BoldYellow(patchsetNum),
		cfg.Server)

	if subject != "" {
		fmt.Printf("Subject: %s\n", utils.Dim(subject))
	}

	// Step 1: Fetch the change
	fmt.Print("Fetching change... ")
	if err := gitFetch(remoteURL, refsPath); err != nil {
		fmt.Println(color.RedString("FAILED"))
		return fmt.Errorf("git fetch failed: %w", err)
	}
	fmt.Println(color.GreenString("SUCCESS"))

	// Step 2: Cherry-pick FETCH_HEAD
	fmt.Print("Cherry-picking... ")
	if err := gitCherryPick("FETCH_HEAD", noCommit, cherryPickNoVerify); err != nil {
		fmt.Println(color.RedString("FAILED"))

		// Check if it's a conflict
		if isCherryPickConflict(err) {
			fmt.Printf("\n%s Cherry-pick has conflicts. Resolve them and then:\n", color.YellowString("⚠"))
			fmt.Println("  • git add <resolved-files>")
			if noCommit {
				fmt.Println("  • git commit (when ready)")
			} else {
				fmt.Println("  • git cherry-pick --continue")
			}
			fmt.Println("  • Or run 'git cherry-pick --abort' to abort")
			return nil // Conflicts are expected, not an error
		}

		return fmt.Errorf("cherry-pick failed: %w", err)
	}
	fmt.Println(color.GreenString("SUCCESS"))

	// Show the result
	if noCommit {
		fmt.Printf("\n%s Change %s has been cherry-picked (not committed)\n",
			color.GreenString("🎉"),
			utils.BoldCyan(changeID))
		fmt.Println("Review the changes and commit when ready:")
		fmt.Println("  git commit")
	} else {
		fmt.Printf("\n%s Change %s has been cherry-picked successfully\n",
			color.GreenString("🎉"),
			utils.BoldCyan(changeID))

		// Show current HEAD info
		if head, err := getGitHead(); err == nil {
			fmt.Printf("HEAD is now at %s\n", utils.Gray(head))
		}
	}
	return nil
}

func isWorkingDirectoryClean() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) == 0
}

func gitCherryPick(ref string, noCommit, noVerify bool) error {
	args := []string{"cherry-pick"}

	if noCommit {
		args = append(args, "--no-commit")
	}

	if noVerify {
		args = append(args, "--no-verify")
	}

	args = append(args, ref)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func isCherryPickConflict(err error) bool {
	// Check if the error is due to conflicts
	if exitError, ok := err.(*exec.ExitError); ok {
		// Git cherry-pick returns exit code 1 for conflicts
		return exitError.ExitCode() == 1
	}
	return false
}
