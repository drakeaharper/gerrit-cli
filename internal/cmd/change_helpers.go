package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/drakeaharper/gerrit-cli/internal/utils"
)

// getStringValue extracts a string from a map, handling float64/int JSON number types.
func getStringValue(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case string:
			return v
		case float64:
			return strconv.FormatFloat(v, 'f', 0, 64)
		case int:
			return strconv.Itoa(v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// getOwnerName extracts the display name from a change's owner field.
func getOwnerName(change map[string]interface{}) string {
	if owner, ok := change["owner"].(map[string]interface{}); ok {
		if name, ok := owner["name"].(string); ok && name != "" {
			return name
		}
		if username, ok := owner["username"].(string); ok && username != "" {
			return username
		}
		if email, ok := owner["email"].(string); ok && email != "" {
			return email
		}
	}
	return "unknown"
}

// getLabelStatus extracts the vote status for any label from a Gerrit change.
// Checks REST format (labels map) first, then SSH format (currentPatchSet.approvals).
func getLabelStatus(change map[string]interface{}, labelName string) string {
	// REST API format: change["labels"][labelName]
	if labels, ok := change["labels"].(map[string]interface{}); ok {
		if labelData, exists := labels[labelName].(map[string]interface{}); exists {
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
	if currentPatchSet, ok := change["currentPatchSet"].(map[string]interface{}); ok {
		if approvals, ok := currentPatchSet["approvals"].([]interface{}); ok {
			for _, approval := range approvals {
				if approvalMap, ok := approval.(map[string]interface{}); ok {
					if approvalType, ok := approvalMap["type"].(string); ok && approvalType == labelName {
						if value, ok := approvalMap["value"]; ok {
							return utils.FormatScore(labelName, value)
						}
					}
				}
			}
		}
	}

	// Label not present
	return utils.Gray("—")
}

// parseSSHChanges parses Gerrit SSH query JSON-lines output into a slice of changes.
func parseSSHChanges(output string) []map[string]interface{} {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var changes []map[string]interface{}

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var change map[string]interface{}
		if err := utils.ParseJSON([]byte(line), &change); err != nil {
			utils.Debugf("Failed to parse line: %s", line)
			continue
		}

		// Skip the stats line
		if _, hasType := change["type"]; hasType {
			continue
		}

		changes = append(changes, change)
	}

	return changes
}

// displayDetailedChanges renders a detailed multi-line view of changes.
func displayDetailedChanges(changes []map[string]interface{}) {
	for i, change := range changes {
		if i > 0 {
			fmt.Println()
		}

		changeNum := getStringValue(change, "_number")
		if changeNum == "" {
			changeNum = getStringValue(change, "number")
		}

		subject := getStringValue(change, "subject")
		status := getStringValue(change, "status")
		updated := getStringValue(change, "updated")
		if updated == "" {
			updated = getStringValue(change, "lastUpdated")
		}

		project := getStringValue(change, "project")
		branch := getStringValue(change, "branch")
		owner := getOwnerName(change)

		fmt.Printf("%s %s\n", utils.BoldCyan("Change:"), utils.BoldWhite(changeNum))
		fmt.Printf("%s %s\n", utils.BoldCyan("Subject:"), subject)
		fmt.Printf("%s %s\n", utils.BoldCyan("Status:"), utils.FormatChangeStatus(status))
		fmt.Printf("%s %s\n", utils.BoldCyan("Project:"), project)
		fmt.Printf("%s %s\n", utils.BoldCyan("Branch:"), branch)
		fmt.Printf("%s %s\n", utils.BoldCyan("Owner:"), owner)
		fmt.Printf("%s %s\n", utils.BoldCyan("Updated:"), utils.FormatTimeAgo(updated))

		// Show review scores if available
		if labels, ok := change["labels"].(map[string]interface{}); ok {
			fmt.Printf("%s ", utils.BoldCyan("Reviews:"))
			var scores []string
			for label, data := range labels {
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
