package gerrit

import (
	"fmt"
	"time"
)

// Account represents a Gerrit user account.
type Account struct {
	AccountID int    `json:"_account_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
}

// DisplayName returns the best available display name for the account.
func (a Account) DisplayName() string {
	if a.Name != "" {
		return a.Name
	}
	if a.Username != "" {
		return a.Username
	}
	if a.Email != "" {
		return a.Email
	}
	return "unknown"
}

// ApprovalInfo represents a single vote on a label (SSH format).
type ApprovalInfo struct {
	Account
	Value int    `json:"value"`
	Date  string `json:"date,omitempty"`
	Type  string `json:"type,omitempty"`
}

// CommitInfo represents a git commit.
type CommitInfo struct {
	Subject string `json:"subject,omitempty"`
	Message string `json:"message,omitempty"`
}

// RevisionInfo represents a single patchset revision.
type RevisionInfo struct {
	Kind   string     `json:"kind,omitempty"`
	Number int        `json:"_number"`
	Commit CommitInfo `json:"commit,omitempty"`
}

// FileInfo describes a changed file in a revision.
type FileInfo struct {
	Status        string `json:"status,omitempty"`
	LinesInserted int    `json:"lines_inserted,omitempty"`
	LinesDeleted  int    `json:"lines_deleted,omitempty"`
	SizeDelta     int    `json:"size_delta,omitempty"`
	Size          int    `json:"size,omitempty"`
	OldPath       string `json:"old_path,omitempty"`
}

// SSHPatchSet represents the currentPatchSet field from SSH query output.
type SSHPatchSet struct {
	Number    int            `json:"number"`
	Revision  string         `json:"revision,omitempty"`
	Ref       string         `json:"ref,omitempty"`
	Approvals []ApprovalInfo `json:"approvals,omitempty"`
}

// Change represents a Gerrit change (CL).
type Change struct {
	// REST API fields
	Number          int                     `json:"_number,omitempty"`
	Project         string                  `json:"project"`
	Branch          string                  `json:"branch"`
	Topic           string                  `json:"topic,omitempty"`
	Subject         string                  `json:"subject"`
	Status          string                  `json:"status"`
	Created         string                  `json:"created,omitempty"`
	Updated         string                  `json:"updated,omitempty"`
	Submitted       string                  `json:"submitted,omitempty"`
	Owner           Account                 `json:"owner"`
	CurrentRevision string                  `json:"current_revision,omitempty"`
	Revisions       map[string]RevisionInfo `json:"revisions,omitempty"`
	Reviewers       map[string][]Account    `json:"reviewers,omitempty"`
	MoreChanges     bool                    `json:"_more_changes,omitempty"`
	URL             string                  `json:"url,omitempty"`

	// Labels kept as untyped map — internal structure varies across Gerrit versions
	Labels map[string]interface{} `json:"labels,omitempty"`

	// SSH-specific fields (mutually exclusive with REST equivalents)
	NumberSSH       int          `json:"number,omitempty"`
	LastUpdated     int64        `json:"lastUpdated,omitempty"`
	CurrentPatchSet *SSHPatchSet `json:"currentPatchSet,omitempty"`
	CommitMessage   string       `json:"commitMessage,omitempty"`
}

// ChangeNumber returns the change number regardless of API source.
func (c Change) ChangeNumber() int {
	if c.Number != 0 {
		return c.Number
	}
	return c.NumberSSH
}

// ChangeNumberStr returns the change number as a string.
func (c Change) ChangeNumberStr() string {
	return fmt.Sprintf("%d", c.ChangeNumber())
}

// UpdatedTime returns the updated timestamp regardless of API source.
func (c Change) UpdatedTime() string {
	if c.Updated != "" {
		return c.Updated
	}
	if c.LastUpdated != 0 {
		return time.Unix(c.LastUpdated, 0).UTC().Format("2006-01-02 15:04:05")
	}
	return ""
}

// CurrentPatchSetNumber returns the current patchset number.
func (c Change) CurrentPatchSetNumber() int {
	if c.CurrentRevision != "" {
		if rev, ok := c.Revisions[c.CurrentRevision]; ok {
			return rev.Number
		}
	}
	if c.CurrentPatchSet != nil {
		return c.CurrentPatchSet.Number
	}
	return 0
}

// CommentInfo represents an inline comment on a change.
type CommentInfo struct {
	ID         string  `json:"id,omitempty"`
	PatchSet   int     `json:"patch_set,omitempty"`
	Line       int     `json:"line,omitempty"`
	Message    string  `json:"message"`
	Updated    string  `json:"updated,omitempty"`
	Author     Account `json:"author"`
	Unresolved bool    `json:"unresolved,omitempty"`
	InReplyTo  string  `json:"in_reply_to,omitempty"`
}

// ChangeMessageInfo represents a change-level message.
type ChangeMessageInfo struct {
	ID      string  `json:"id,omitempty"`
	Author  Account `json:"author"`
	Message string  `json:"message"`
	Date    string  `json:"date,omitempty"`
	Tag     string  `json:"tag,omitempty"`
}
