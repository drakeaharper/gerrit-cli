package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	checkoutFetch bool
	noVerify      bool
)

var fetchCmd = &cobra.Command{
	Use:   "fetch <change-id> [patchset]",
	Short: "Fetch a change",
	Long:  `Fetch a change and checkout to FETCH_HEAD. If patchset is not specified, fetches the current patch set.`,
	Args:  cobra.RangeArgs(1, 2),
	Run:   runFetch,
}

func init() {
	fetchCmd.Flags().BoolVarP(&checkoutFetch, "checkout", "c", true, "Checkout to FETCH_HEAD after fetching")
	fetchCmd.Flags().BoolVar(&noVerify, "no-verify", false, "Skip git hooks during checkout")
}

func runFetch(cmd *cobra.Command, args []string) {
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

	// Check if we're in a git repository
	if !isGitRepository() {
		utils.ExitWithError(fmt.Errorf("not in a git repository"))
	}

	utils.Debugf("Fetching change %s patchset %s", changeID, patchset)

	// Get change details to build the fetch ref
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

	// Build the refs path
	refsPath := fmt.Sprintf("refs/changes/%s/%s/%s",
		getChangePrefix(changeID),
		changeID,
		patchsetNum)

	utils.Debugf("Fetching from refs: %s", refsPath)

	// Get git remote URL for the server
	remoteURL := buildRemoteURL(cfg)
	
	fmt.Printf("Fetching change %s (patchset %s) from %s...\n", 
		utils.BoldCyan(changeID), 
		utils.BoldYellow(patchsetNum),
		cfg.Server)

	// Execute git fetch
	if err := gitFetch(remoteURL, refsPath); err != nil {
		utils.ExitWithError(fmt.Errorf("git fetch failed: %w", err))
	}

	fmt.Printf("%s Successfully fetched change\n", color.GreenString("✓"))

	// Checkout to FETCH_HEAD if requested
	if checkoutFetch {
		fmt.Print("Checking out to FETCH_HEAD... ")
		if err := gitCheckout("FETCH_HEAD", noVerify); err != nil {
			fmt.Println(color.RedString("FAILED"))
			utils.ExitWithError(fmt.Errorf("checkout failed: %w", err))
		}
		fmt.Println(color.GreenString("SUCCESS"))
		
		// Show current HEAD info
		if head, err := getGitHead(); err == nil {
			fmt.Printf("HEAD is now at %s\n", utils.Gray(head))
		}
	}
	
	fmt.Printf("\n%s Change %s is ready for review\n", 
		color.GreenString("🎉"), 
		utils.BoldCyan(changeID))
	
	if !checkoutFetch {
		fmt.Println("Use 'git checkout FETCH_HEAD' to switch to the fetched change")
	}
}

func getChangeForFetch(cfg *config.Config, changeID string) (map[string]interface{}, error) {
	// Try REST API first, fall back to SSH
	client := gerrit.NewRESTClient(cfg)
	change, err := client.GetChange(changeID)
	if err != nil {
		utils.Debugf("REST API failed: %v", err)
		// Fall back to SSH
		sshClient := gerrit.NewSSHClient(cfg)
		output, err := sshClient.ExecuteCommand(fmt.Sprintf("query --format=JSON --current-patch-set %s", changeID))
		if err != nil {
			return nil, err
		}
		
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			
			var changeData map[string]interface{}
			if err := utils.ParseJSON([]byte(line), &changeData); err != nil {
				continue
			}
			
			// Skip the stats line
			if _, hasType := changeData["type"]; hasType {
				continue
			}
			
			return changeData, nil
		}
		return nil, fmt.Errorf("no valid change data found")
	}
	
	return change, nil
}

func getCurrentPatchsetNumber(change map[string]interface{}) string {
	// Try REST API format first
	if revisions, ok := change["revisions"].(map[string]interface{}); ok {
		currentRevision := getStringValue(change, "current_revision")
		if currentRevision != "" {
			if rev, ok := revisions[currentRevision].(map[string]interface{}); ok {
				return getStringValue(rev, "_number")
			}
		}
	}
	
	// Try SSH API format
	if currentPatchSet, ok := change["currentPatchSet"].(map[string]interface{}); ok {
		return getStringValue(currentPatchSet, "number")
	}
	
	// Try direct number field
	return getStringValue(change, "number")
}

func getChangePrefix(changeID string) string {
	// Gerrit uses the last two digits of the change number for the prefix
	if len(changeID) >= 2 {
		return changeID[len(changeID)-2:]
	}
	return "00"
}

func buildRemoteURL(cfg *config.Config) string {
	// Prefer SSH for git operations (more reliable with SSH keys)
	if cfg.Project != "" {
		return fmt.Sprintf("ssh://%s@%s:%d/%s", cfg.User, cfg.Server, cfg.Port, cfg.Project)
	}
	return fmt.Sprintf("ssh://%s@%s:%d", cfg.User, cfg.Server, cfg.Port)
}

func isGitRepository() bool {
	_, err := os.Stat(".git")
	if err == nil {
		return true
	}
	
	// Check if we're in a subdirectory of a git repo
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

func gitFetch(remoteURL, refsPath string) error {
	cmd := exec.Command("git", "fetch", remoteURL, refsPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitCheckout(ref string, noVerify bool) error {
	args := []string{"checkout"}
	if noVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, ref)
	
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getGitHead() (string, error) {
	cmd := exec.Command("git", "log", "--oneline", "-1", "--no-decorate")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}