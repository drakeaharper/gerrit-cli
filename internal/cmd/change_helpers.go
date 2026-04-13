package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
)

// getLabelStatus extracts the vote status for any label from a Gerrit change.
// Checks REST format (labels map) first, then SSH format (currentPatchSet.approvals).
func getLabelStatus(change gerrit.Change, labelName string) string {
	// REST API format: labels[labelName] is a LabelInfo-like object
	if change.Labels != nil {
		if labelData, exists := change.Labels[labelName].(map[string]interface{}); exists {
			// Check approved map (highest positive vote)
			if approved, hasApproved := labelData["approved"].(map[string]interface{}); hasApproved {
				if value, ok := approved["value"]; ok {
					return utils.FormatScore(labelName, value)
				}
				// approved present but no value field — infer positive
				return utils.FormatScore(labelName, 1)
			}
			// Check rejected map (lowest negative vote)
			if rejected, hasRejected := labelData["rejected"].(map[string]interface{}); hasRejected {
				if value, ok := rejected["value"]; ok {
					return utils.FormatScore(labelName, value)
				}
				// rejected present but no value field — infer negative
				return utils.FormatScore(labelName, -1)
			}
			// Check individual votes in "all" array
			if all, hasAll := labelData["all"].([]interface{}); hasAll && len(all) > 0 {
				hasVote := false
				maxScore := -3
				for _, vote := range all {
					if voteMap, ok := vote.(map[string]interface{}); ok {
						if value, ok := voteMap["value"]; ok {
							if score, ok := value.(float64); ok {
								hasVote = true
								if int(score) > maxScore {
									maxScore = int(score)
								}
							}
						}
					}
				}
				if hasVote {
					return utils.FormatScore(labelName, maxScore)
				}
			}
			// Label exists but no votes
			return utils.Gray("0")
		}
	}

	// SSH API format: currentPatchSet.approvals
	if change.CurrentPatchSet != nil {
		for _, approval := range change.CurrentPatchSet.Approvals {
			if approval.Type == labelName {
				return utils.FormatScore(labelName, approval.Value)
			}
		}
	}

	// Label not present
	return utils.Gray("—")
}

// getAuthorName extracts a display name from an untyped account map.
// Used for label internals where the structure varies across Gerrit versions.
func getAuthorName(author map[string]interface{}) string {
	if name, ok := author["name"].(string); ok && name != "" {
		return name
	}
	if username, ok := author["username"].(string); ok && username != "" {
		return username
	}
	if email, ok := author["email"].(string); ok && email != "" {
		return email
	}
	return "unknown"
}

// parseSSHChanges parses Gerrit SSH query JSON-lines output into a slice of changes.
func parseSSHChanges(output string) []gerrit.Change {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var changes []gerrit.Change

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Peek to skip the stats line
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			utils.Debugf("Failed to parse line: %s", line)
			continue
		}
		if _, hasType := raw["type"]; hasType {
			continue
		}

		var change gerrit.Change
		if err := json.Unmarshal([]byte(line), &change); err != nil {
			utils.Debugf("Failed to parse change: %s", line)
			continue
		}

		changes = append(changes, change)
	}

	return changes
}

// parseSSHChangeDetail parses a single change from SSH JSON-lines output.
func parseSSHChangeDetail(output string) (*gerrit.Change, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			utils.Debugf("Failed to parse line: %s", line)
			continue
		}
		if _, hasType := raw["type"]; hasType {
			continue
		}

		var change gerrit.Change
		if err := json.Unmarshal([]byte(line), &change); err != nil {
			continue
		}

		return &change, nil
	}

	return nil, fmt.Errorf("no valid change data found")
}

// displayDetailedChanges renders a detailed multi-line view of changes.
func displayDetailedChanges(changes []gerrit.Change) {
	for i, change := range changes {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("%s %s\n", utils.BoldCyan("Change:"), utils.BoldWhite(change.ChangeNumberStr()))
		fmt.Printf("%s %s\n", utils.BoldCyan("Subject:"), change.Subject)
		fmt.Printf("%s %s\n", utils.BoldCyan("Status:"), utils.FormatChangeStatus(change.Status))
		fmt.Printf("%s %s\n", utils.BoldCyan("Project:"), change.Project)
		fmt.Printf("%s %s\n", utils.BoldCyan("Branch:"), change.Branch)
		fmt.Printf("%s %s\n", utils.BoldCyan("Owner:"), change.Owner.DisplayName())
		fmt.Printf("%s %s\n", utils.BoldCyan("Updated:"), utils.FormatTimeAgo(change.UpdatedTime()))

		// Show review scores if available
		if len(change.Labels) > 0 {
			fmt.Printf("%s ", utils.BoldCyan("Reviews:"))
			var scores []string
			for label, data := range change.Labels {
				if labelData, ok := data.(map[string]interface{}); ok {
					if approved, ok := labelData["approved"].(map[string]interface{}); ok {
						if value, ok := approved["value"]; ok {
							scores = append(scores, fmt.Sprintf("%s:%s", label, utils.FormatScore(label, value)))
						}
					} else if rejected, ok := labelData["rejected"].(map[string]interface{}); ok {
						if value, ok := rejected["value"]; ok {
							scores = append(scores, fmt.Sprintf("%s:%s", label, utils.FormatScore(label, value)))
						}
					}
				}
			}
			if len(scores) > 0 {
				fmt.Println(strings.Join(scores, " "))
			} else {
				fmt.Println(utils.Gray("none"))
			}
		}
	}
}
